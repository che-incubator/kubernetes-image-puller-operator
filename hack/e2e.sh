#!/bin/bash
#
# Copyright (c) 2019-2026 Red Hat, Inc.
# This program and the accompanying materials are made
# available under the terms of the Eclipse Public License 2.0
# which is available at https://www.eclipse.org/legal/epl-2.0/
#
# SPDX-License-Identifier: EPL-2.0
#
# Contributors:
#   Red Hat, Inc. - initial API and implementation
#
# Verifies a Kubernetes Image Puller Operator release against the cluster in the
# current kubeconfig. Used for the Kubernetes and OpenShift verification steps
# of the release process (README, "Releasing a new version of the operator to
# OperatorHub").
#
# The operator is either deployed from the manifests in this checkout
# (--install-method deploy) or installed from a published OLM bundle
# (--install-method olm). The script then exercises the full lifecycle of a
# KubernetesImagePuller resource, including the update and delete paths and the
# validating webhook.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Make the tools installed by the Makefile (operator-sdk, kustomize) available.
export PATH="${REPO_ROOT}/bin:${PATH}"

NAMESPACE="kubernetes-image-puller-operator"
INSTALL_METHOD="deploy"
BUNDLE_IMAGE=""
CR_NAME="image-puller"
CR_IMAGES=""
TIMEOUT_SECONDS=600
ARTIFACT_DIR="${ARTIFACT_DIR:-}"

# Defaults from pkg/defaults. Used when the CR does not override them.
DEFAULT_CONFIGMAP_NAME="k8s-image-puller"
DEFAULT_DEPLOYMENT_NAME="kubernetes-image-puller"
DEFAULT_DAEMONSET_NAME="kubernetes-image-puller"

usage() {
  cat <<EOF
Usage: $0 [options]

Options:
  --install-method deploy|olm   How to install the operator (default: deploy)
  --namespace NAMESPACE         Namespace to install into (default: ${NAMESPACE})
  --bundle-image IMAGE          Bundle image for --install-method olm
  --cr-name NAME                Name of the KubernetesImagePuller CR (default: ${CR_NAME})
  --images LIST                 Value for spec.images, overriding the sample CR
  --timeout SECONDS             Timeout for each wait (default: ${TIMEOUT_SECONDS})
  --artifact-dir DIR            Directory for diagnostics on failure
  -h, --help                    Show this help
EOF
}

log() {
  echo "[INFO] $*"
}

fail() {
  echo "[ERROR] $*" >&2
  exit 1
}

parse_args() {
  while [[ "$#" -gt 0 ]]; do
    case $1 in
      '--install-method') INSTALL_METHOD="$2"; shift 2;;
      '--namespace') NAMESPACE="$2"; shift 2;;
      '--bundle-image') BUNDLE_IMAGE="$2"; shift 2;;
      '--cr-name') CR_NAME="$2"; shift 2;;
      '--images') CR_IMAGES="$2"; shift 2;;
      '--timeout') TIMEOUT_SECONDS="$2"; shift 2;;
      '--artifact-dir') ARTIFACT_DIR="$2"; shift 2;;
      '--help'|'-h') usage; exit 0;;
      *) usage; fail "unknown option: $1";;
    esac
  done

  case ${INSTALL_METHOD} in
    deploy|olm) ;;
    *) fail "--install-method must be 'deploy' or 'olm'";;
  esac

  if [[ ${INSTALL_METHOD} == "olm" && -z ${BUNDLE_IMAGE} ]]; then
    fail "--bundle-image is required with --install-method olm"
  fi
}

detect_platform() {
  if kubectl api-resources --api-group='route.openshift.io' 2>/dev/null | grep -q '^routes'; then
    echo "openshift"
  else
    echo "kubernetes"
  fi
}

dump_diagnostics() {
  [[ -n ${ARTIFACT_DIR} ]] || return 0
  local dir
  dir="${ARTIFACT_DIR}/e2e-$(date +%Y%m%dT%H%M%S)"
  mkdir -p "${dir}"
  {
    kubectl get all -n "${NAMESPACE}" -o wide
    kubectl get kubernetesimagepullers -n "${NAMESPACE}" -o yaml
    kubectl get events -n "${NAMESPACE}" --sort-by=.lastTimestamp
    kubectl get validatingwebhookconfiguration -o name | grep -i imagepull || true
  } >"${dir}/cluster.txt" 2>&1 || true
  kubectl logs -n "${NAMESPACE}" -l name=kubernetes-image-puller-operator \
    --all-containers --tail=300 >"${dir}/operator.log" 2>&1 || true
  kubectl logs -n "${NAMESPACE}" -l app=kubernetes-image-puller \
    --all-containers --tail=300 >"${dir}/image-puller.log" 2>&1 || true
  log "diagnostics written to ${dir}"
}

on_error() {
  local code=$?
  if [[ ${code} -ne 0 ]]; then
    echo "[ERROR] verification failed (exit ${code})" >&2
    dump_diagnostics
  fi
  exit "${code}"
}

preflight() {
  command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
  kubectl cluster-info >/dev/null 2>&1 || fail "no reachable Kubernetes cluster in the current kubeconfig"
  [[ -z ${ARTIFACT_DIR} ]] || mkdir -p "${ARTIFACT_DIR}"

  # A second installation registers a cluster-scoped webhook whose cache is
  # scoped to its own namespace, which makes every CR creation fail. Refuse to
  # run on top of an existing installation instead of producing confusing
  # errors.
  if kubectl get validatingwebhookconfiguration -o name 2>/dev/null | grep -q 'vkubernetesimagepuller'; then
    fail "a Kubernetes Image Puller webhook is already installed; remove the existing installation first"
  fi
  if kubectl get customresourcedefinitions.apiextensions.k8s.io kubernetesimagepullers.che.eclipse.org >/dev/null 2>&1; then
    fail "the KubernetesImagePuller CRD already exists; verify on a clean cluster"
  fi
}

wait_for() {
  local description=$1
  local command=$2
  local deadline=$((SECONDS + TIMEOUT_SECONDS))
  log "waiting for ${description}"
  until eval "${command}" >/dev/null 2>&1; do
    if ((SECONDS >= deadline)); then
      fail "timed out after ${TIMEOUT_SECONDS}s waiting for ${description}"
    fi
    sleep 3
  done
}

cr_field() {
  local field=$1
  local fallback=$2
  local value
  value="$(kubectl get kubernetesimagepuller "${CR_NAME}" -n "${NAMESPACE}" -o jsonpath="{.spec.${field}}" 2>/dev/null || true)"
  echo "${value:-${fallback}}"
}

install_operator() {
  kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f - >/dev/null

  case ${INSTALL_METHOD} in
    deploy)
      log "deploying the operator from ${REPO_ROOT} ($(detect_platform))"
      (cd "${REPO_ROOT}" && make deploy)
      ;;
    olm)
      command -v operator-sdk >/dev/null 2>&1 || fail "operator-sdk is required for --install-method olm"
      if ! kubectl get crd subscriptions.operators.coreos.com >/dev/null 2>&1; then
        log "installing OLM"
        operator-sdk olm install --version v0.28.0 --timeout 5m
      fi
      log "installing bundle ${BUNDLE_IMAGE}"
      operator-sdk run bundle "${BUNDLE_IMAGE}" -n "${NAMESPACE}" --timeout 8m
      wait_for "the ClusterServiceVersion to succeed" \
        "[[ \$(kubectl get csv -n ${NAMESPACE} -o jsonpath='{.items[*].status.phase}') == *Succeeded* ]]"
      ;;
  esac

  wait_for "the operator deployment to become available" \
    "[[ \$(kubectl get deployment -n ${NAMESPACE} -l name=kubernetes-image-puller-operator -o jsonpath='{.items[0].status.availableReplicas}') -ge 1 ]]"
}

create_cr() {
  local manifest
  if [[ -n ${CR_IMAGES} ]]; then
    manifest="$(mktemp)"
    cat >"${manifest}" <<EOF
apiVersion: che.eclipse.org/v1alpha1
kind: KubernetesImagePuller
metadata:
  name: ${CR_NAME}
spec:
  daemonsetName: k8s-image-puller
  images: "${CR_IMAGES}"
EOF
  else
    manifest="${REPO_ROOT}/config/samples/che_v1alpha1_kubernetesimagepuller.yaml"
  fi

  log "creating KubernetesImagePuller ${CR_NAME}"
  kubectl apply -f "${manifest}" -n "${NAMESPACE}" >/dev/null
  [[ -z ${CR_IMAGES} ]] || rm -f "${manifest}"

  wait_for "the KubernetesImagePuller to become Ready" \
    "[[ \$(kubectl get kubernetesimagepuller ${CR_NAME} -n ${NAMESPACE} -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') == True ]]"
}

verify_owned_resources() {
  local configmap deployment daemonset key missing=""
  configmap="$(cr_field configMapName ${DEFAULT_CONFIGMAP_NAME})"
  deployment="$(cr_field deploymentName ${DEFAULT_DEPLOYMENT_NAME})"
  daemonset="$(cr_field daemonsetName ${DEFAULT_DAEMONSET_NAME})"

  log "checking ConfigMap ${configmap}"
  for key in IMAGES CACHING_INTERVAL_HOURS CACHING_MEMORY_REQUEST CACHING_MEMORY_LIMIT \
    DAEMONSET_NAME NODE_SELECTOR KIP_IMAGE NAMESPACE; do
    if [[ -z $(kubectl get configmap "${configmap}" -n "${NAMESPACE}" -o jsonpath="{.data.${key}}") ]]; then
      missing="${missing} ${key}"
    fi
  done
  [[ -z ${missing} ]] || fail "ConfigMap ${configmap} is missing keys:${missing}"

  wait_for "the image puller deployment ${deployment} to become available" \
    "[[ \$(kubectl get deployment ${deployment} -n ${NAMESPACE} -o jsonpath='{.status.availableReplicas}') -ge 1 ]]"

  log "checking DaemonSet ${daemonset}"
  wait_for "the DaemonSet to be created" \
    "kubectl get daemonset ${daemonset} -n ${NAMESPACE}"
  wait_for "the DaemonSet to schedule a pod" \
    "[[ \$(kubectl get daemonset ${daemonset} -n ${NAMESPACE} -o jsonpath='{.status.currentNumberScheduled}') -ge 1 ]]"
  wait_for "the DaemonSet pods to become ready" \
    "[[ \$(kubectl get daemonset ${daemonset} -n ${NAMESPACE} -o jsonpath='{.status.numberReady}') -ge 1 ]]"
}

verify_update() {
  local configmap deployment before after
  configmap="$(cr_field configMapName ${DEFAULT_CONFIGMAP_NAME})"
  deployment="$(cr_field deploymentName ${DEFAULT_DEPLOYMENT_NAME})"

  before="$(kubectl get deployment "${deployment}" -n "${NAMESPACE}" \
    -o jsonpath='{.spec.template.metadata.annotations.che\.eclipse\.org/configmap-hash}')"

  log "updating the CR and checking the change propagates"
  kubectl patch kubernetesimagepuller "${CR_NAME}" -n "${NAMESPACE}" --type=merge \
    -p '{"spec":{"cachingIntervalHours":"7","cachingMemoryLimit":"27Mi"}}' >/dev/null

  wait_for "the ConfigMap to reflect cachingIntervalHours=7" \
    "[[ \$(kubectl get configmap ${configmap} -n ${NAMESPACE} -o jsonpath='{.data.CACHING_INTERVAL_HOURS}') == 7 ]]"
  wait_for "the ConfigMap to reflect cachingMemoryLimit=27Mi" \
    "[[ \$(kubectl get configmap ${configmap} -n ${NAMESPACE} -o jsonpath='{.data.CACHING_MEMORY_LIMIT}') == 27Mi ]]"

  wait_for "the image puller deployment to pick up the new ConfigMap hash" \
    "[[ \"\$(kubectl get deployment ${deployment} -n ${NAMESPACE} -o jsonpath='{.spec.template.metadata.annotations.che\\.eclipse\\.org/configmap-hash}')\" != '${before}' ]]"
  after="$(kubectl get deployment "${deployment}" -n "${NAMESPACE}" \
    -o jsonpath='{.spec.template.metadata.annotations.che\.eclipse\.org/configmap-hash}')"
  [[ ${before} != "${after}" ]] || fail "the deployment was not updated when the ConfigMap changed"

  wait_for "the image puller deployment to roll out the new configuration" \
    "[[ \$(kubectl get deployment ${deployment} -n ${NAMESPACE} -o jsonpath='{.status.availableReplicas}') -ge 1 ]]"
}

verify_webhook() {
  local manifest output
  manifest="$(mktemp)"
  cat >"${manifest}" <<EOF
apiVersion: che.eclipse.org/v1alpha1
kind: KubernetesImagePuller
metadata:
  name: ${CR_NAME}-second
spec:
  daemonsetName: k8s-image-puller-second
  images: "che-code=quay.io/che-incubator/che-code:next"
EOF

  log "checking the validating webhook rejects a second KubernetesImagePuller"
  if output="$(kubectl apply -f "${manifest}" -n "${NAMESPACE}" 2>&1)"; then
    rm -f "${manifest}"
    kubectl delete kubernetesimagepuller "${CR_NAME}-second" -n "${NAMESPACE}" --ignore-not-found >/dev/null
    fail "the webhook allowed a second KubernetesImagePuller in the same namespace"
  fi
  rm -f "${manifest}"

  if ! grep -q "only one KubernetesImagePuller is allowed per namespace" <<<"${output}"; then
    fail "a second KubernetesImagePuller was rejected for an unexpected reason: ${output}"
  fi
}

verify_delete() {
  local configmap deployment daemonset resource
  configmap="$(cr_field configMapName ${DEFAULT_CONFIGMAP_NAME})"
  deployment="$(cr_field deploymentName ${DEFAULT_DEPLOYMENT_NAME})"
  daemonset="$(cr_field daemonsetName ${DEFAULT_DAEMONSET_NAME})"

  log "deleting the CR and checking the owned resources are removed"
  kubectl delete kubernetesimagepuller "${CR_NAME}" -n "${NAMESPACE}" --timeout=120s >/dev/null

  wait_for "the KubernetesImagePuller to be deleted" \
    "! kubectl get kubernetesimagepuller ${CR_NAME} -n ${NAMESPACE}"
  for resource in "configmap/${configmap}" "deployment/${deployment}" "daemonset/${daemonset}"; do
    wait_for "${resource} to be removed" "! kubectl get ${resource} -n ${NAMESPACE}"
  done
}

main() {
  parse_args "$@"
  trap on_error EXIT
  preflight
  install_operator
  create_cr
  verify_owned_resources
  verify_update
  verify_webhook
  verify_delete
  trap - EXIT
  log "verification succeeded on $(detect_platform) with install method '${INSTALL_METHOD}'"
}

main "$@"
