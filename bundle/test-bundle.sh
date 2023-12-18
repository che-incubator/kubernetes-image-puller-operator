#!/bin/bash
#
# Copyright (c) 2019-2021 Red Hat, Inc.
# This program and the accompanying materials are made
# available under the terms of the Eclipse Public License 2.0
# which is available at https://www.eclipse.org/legal/epl-2.0/
#
# SPDX-License-Identifier: EPL-2.0
#
# Contributors:
#   Red Hat, Inc. - initial API and implementation
#

set -e

OPERATOR_REPO="$(dirname "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")")"

export KIP_ROOT_DIR=/tmp/kubernetes-imagepuller-operator
export SOURCE_DIR=${KIP_ROOT_DIR}/sources
export CATALOG_DIR=${KIP_ROOT_DIR}/olm-catalog/stable
export BUNDLE_DIR=${KIP_ROOT_DIR}/bundle
export CA_DIR=${KIP_ROOT_DIR}/certificates
export NAMESPACE=kubernetes-image-puller-operator

# Images names in the OpenShift registry
export REGISTRY_BUNDLE_IMAGE_NAME="kubernetes-imagepuller-bundle"
export REGISTRY_CATALOG_IMAGE_NAME="kubernetes-imagepuller-catalog"
export REGISTRY_OPERATOR_IMAGE_NAME="kubernetes-imagepuller-operator"

# Images
unset OPERATOR_IMAGE
unset BUNDLE_IMAGE
unset CATALOG_IMAGE

init() {
  while [[ "$#" -gt 0 ]]; do
    case $1 in
      '--help'|'-h') usage; exit;;
    esac
    shift 1
  done

  rm -rf ${KIP_ROOT_DIR}
  mkdir -p ${CATALOG_DIR}
  mkdir -p ${BUNDLE_DIR}
  mkdir -p ${CA_DIR}
  mkdir -p ${SOURCE_DIR}
}

usage () {
  echo "Deploy Kubernetes imagepuller from sources"
  echo
	echo "Example:"
	echo -e "\t$0"
}

exposeOpenShiftRegistry() {
  oc patch configs.imageregistry.operator.openshift.io/cluster --patch '{"spec":{"defaultRoute":true}}' --type=merge
  sleep 10s
  REGISTRY_HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')

  BUNDLE_IMAGE="${REGISTRY_HOST}/${NAMESPACE}/${REGISTRY_BUNDLE_IMAGE_NAME}:latest"
  echo "[INFO] Bundle image: ${BUNDLE_IMAGE}"

  OPERATOR_IMAGE="${REGISTRY_HOST}/${NAMESPACE}/${REGISTRY_OPERATOR_IMAGE_NAME}:latest"
  echo "[INFO] Operator image: ${OPERATOR_IMAGE}"

  CATALOG_IMAGE="${REGISTRY_HOST}/${NAMESPACE}/${REGISTRY_CATALOG_IMAGE_NAME}:latest"
  echo "[INFO] Catalog image: ${CATALOG_IMAGE}"

  oc get secret -n openshift-ingress  router-certs-default -o go-template='{{index .data "tls.crt"}}' | base64 -d > ${CA_DIR}/ca.crt

  oc delete configmap openshift-registry --ignore-not-found=true -n openshift-config
  oc create configmap openshift-registry -n openshift-config --from-file=${REGISTRY_HOST}=${CA_DIR}/ca.crt

  oc patch image.config.openshift.io/cluster --patch '{"spec":{"additionalTrustedCA":{"name":"openshift-registry"}}}' --type=merge

  oc policy add-role-to-user system:image-builder system:anonymous -n "${NAMESPACE}"
  oc policy add-role-to-user system:image-builder system:unauthenticated -n "${NAMESPACE}"
}

buildOperatorFromSources() {
  oc delete buildconfigs ${REGISTRY_OPERATOR_IMAGE_NAME} --ignore-not-found=true -n "${NAMESPACE}"
  oc delete imagestreamtag ${REGISTRY_OPERATOR_IMAGE_NAME}:latest --ignore-not-found=true -n "${NAMESPACE}"

  # Move Dockerfile to the root directory
  cp -r "${OPERATOR_REPO}"/* "${SOURCE_DIR}"
  cp -r ${OPERATOR_REPO}/build/* "${SOURCE_DIR}"

  oc new-build --binary --strategy docker --name "${REGISTRY_OPERATOR_IMAGE_NAME}" -n "${NAMESPACE}"
  oc start-build "${REGISTRY_OPERATOR_IMAGE_NAME}" --from-dir "${SOURCE_DIR}" -n "${NAMESPACE}" --wait
}

buildBundleFromSources() {
  cp -r $(make bundle-path)/* ${BUNDLE_DIR}
  mv ${BUNDLE_DIR}/bundle.Dockerfile ${BUNDLE_DIR}/Dockerfile

  # Set operator image from the registry
  yq -riY '.spec.install.spec.deployments[0].spec.template.spec.containers[] |= (select(.name == "kubernetes-image-puller-operator") .image |= "'${OPERATOR_IMAGE}'")' ${BUNDLE_DIR}/manifests/kubernetes-imagepuller-operator.clusterserviceversion.yaml

  oc delete buildconfigs ${REGISTRY_BUNDLE_IMAGE_NAME} --ignore-not-found=true -n "${NAMESPACE}"
  oc delete imagestreamtag ${REGISTRY_BUNDLE_IMAGE_NAME}:latest --ignore-not-found=true -n "${NAMESPACE}"

  oc new-build --binary --strategy docker --name "${REGISTRY_BUNDLE_IMAGE_NAME}" -n "${NAMESPACE}"
  oc start-build "${REGISTRY_BUNDLE_IMAGE_NAME}" --from-dir ${BUNDLE_DIR} -n "${NAMESPACE}" --wait
}

buildCatalogFromSources() {
  cat > ${CATALOG_DIR}/package.yaml <<EOF
schema: olm.package
name: kubernetes-imagepuller-operator
defaultChannel: stable
EOF

  cat > ${CATALOG_DIR}/channel.yaml <<EOF
schema: olm.channel
package: kubernetes-imagepuller-operator
name: stable
entries:
  - name: $(make bundle-name)
EOF

  PARENT_CATALOG_DIR=$(dirname "${CATALOG_DIR}")
  make bundle-render CATALOG_DIR="${CATALOG_DIR}" BUNDLE_IMG="${BUNDLE_IMAGE}"
  cp "${OPERATOR_REPO}/olm-catalog/index.Dockerfile" "${PARENT_CATALOG_DIR}/Dockerfile"
  sed -i 's|olm-catalog/stable|stable|g' "${PARENT_CATALOG_DIR}/Dockerfile"

  oc delete buildconfigs ${REGISTRY_CATALOG_IMAGE_NAME} --ignore-not-found=true -n "${NAMESPACE}"
  oc delete imagestreamtag ${REGISTRY_CATALOG_IMAGE_NAME}:latest --ignore-not-found=true -n "${NAMESPACE}"

  oc new-build --binary --strategy docker --name "${REGISTRY_CATALOG_IMAGE_NAME}" -n "${NAMESPACE}"
  oc start-build "${REGISTRY_CATALOG_IMAGE_NAME}" --from-dir "${PARENT_CATALOG_DIR}" -n "${NAMESPACE}" --wait
}

createKIPCatalogFromSources() {
  buildOperatorFromSources
  buildBundleFromSources
  buildCatalogFromSources

  kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: kubernetes-imagepuller-operator
  namespace: ${NAMESPACE}
spec: {}
EOF

  kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: kubernetes-imagepuller-operator
  namespace: ${NAMESPACE}
spec:
  image: ${CATALOG_IMAGE}
  sourceType: grpc
  updateStrategy:
    registryPoll:
      interval: 15m
EOF
}

run() {
  oc create namespace ${NAMESPACE}

  exposeOpenShiftRegistry
  createKIPCatalogFromSources

  kubectl apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kubernetes-imagepuller-operator
  namespace: ${NAMESPACE}
spec:
  channel: stable
  installPlanApproval: Auto
  name: kubernetes-imagepuller-operator
  source: kubernetes-imagepuller-operator
  sourceNamespace: ${NAMESPACE}
EOF
}

init "$@"
[[ ${VERBOSE} == 1 ]] && set -x

pushd "${OPERATOR_REPO}" >/dev/null
run
popd >/dev/null

