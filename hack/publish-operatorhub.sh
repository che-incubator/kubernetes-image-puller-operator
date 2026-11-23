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
# Publishes a released bundle to OperatorHub by copying it into the
# community-operators and community-operators-prod repositories, committing it
# on a branch in a fork and opening (or updating) the two pull requests.
#
# Run this from the release branch of this repository, after the bundle for the
# requested version has been committed there.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

VERSION=""
FORK_OWNER=""
WORK_DIR="${WORK_DIR:-${REPO_ROOT}/.operatorhub}"
DRY_RUN="false"
GIT_NAME=""
GIT_EMAIL=""
YQ_PATH=""

UPSTREAM_COMMUNITY_OPERATORS="k8s-operatorhub/community-operators"
UPSTREAM_COMMUNITY_OPERATORS_PROD="redhat-openshift-ecosystem/community-operators-prod"

usage() {
  cat <<'EOF'
Usage: hack/publish-operatorhub.sh --version X.Y.Z [options]

Options:
  --version X.Y.Z     Version to publish (must match the bundle in this checkout)
  --fork-owner USER   GitHub user that owns the forks (default: current gh user)
  --work-dir DIR      Where to clone the OperatorHub repositories
  --git-name NAME     Author name for the commit (default: git config user.name)
  --git-email EMAIL   Author email for the commit (default: git config user.email)
  --dry-run           Prepare the commits but do not push or open pull requests
  -h, --help          Show this help
EOF
}

log() {
  echo "[INFO] $*" >&2
}

fail() {
  echo "[ERROR] $*" >&2
  exit 1
}

parse_args() {
  while [[ "$#" -gt 0 ]]; do
    case $1 in
      '--version') VERSION="${2#v}"; shift 2;;
      '--fork-owner') FORK_OWNER="$2"; shift 2;;
      '--work-dir') WORK_DIR="$2"; shift 2;;
      '--git-name') GIT_NAME="$2"; shift 2;;
      '--git-email') GIT_EMAIL="$2"; shift 2;;
      '--dry-run') DRY_RUN="true"; shift;;
      '--help'|'-h') usage; exit 0;;
      *) usage; fail "unknown option: $1";;
    esac
  done

  [[ -n ${VERSION} ]] || { usage; fail "--version is required"; }
  [[ ${VERSION} =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "--version must look like X.Y.Z"

  if [[ -z ${FORK_OWNER} ]]; then
    command -v gh >/dev/null 2>&1 || fail "gh is required (or pass --fork-owner)"
    FORK_OWNER="$(gh api user --jq .login)"
  fi

  GIT_NAME="${GIT_NAME:-$(git config --global user.name || true)}"
  GIT_EMAIL="${GIT_EMAIL:-$(git config --global user.email || true)}"
  [[ -n ${GIT_NAME} ]] || fail "no git author name; pass --git-name"
  [[ -n ${GIT_EMAIL} ]] || fail "no git author email; pass --git-email"
}

preflight() {
  command -v git >/dev/null 2>&1 || fail "git is required"
  command -v gh >/dev/null 2>&1 || fail "gh is required"
  gh auth status >/dev/null 2>&1 || fail "gh is not authenticated"

  local bundle_version
  bundle_version="$(cd "${REPO_ROOT}" && make bundle-version | tail -1)"
  [[ ${bundle_version} == "${VERSION}" ]] || \
    fail "this checkout contains bundle version '${bundle_version}', not '${VERSION}'. Check out the ${VERSION}-release branch."

  YQ_PATH="$(cd "${REPO_ROOT}" && make yq-path 2>/dev/null | tail -1)"
  [[ -x ${YQ_PATH} ]] || fail "unable to locate the repository yq (run 'make download-yq')"

  local branch
  branch="$(cd "${REPO_ROOT}" && git rev-parse --abbrev-ref HEAD)"
  log "publishing bundle ${VERSION} from branch ${branch}"

  mkdir -p "${WORK_DIR}"
}

ensure_fork() {
  local upstream=$1
  local name=${upstream#*/}
  if gh repo view "${FORK_OWNER}/${name}" >/dev/null 2>&1; then
    log "fork ${FORK_OWNER}/${name} already exists"
  else
    log "creating fork ${FORK_OWNER}/${name}"
    (cd "${WORK_DIR}" && gh repo fork "${upstream}" --clone=false >/dev/null)
    # Fork creation is asynchronous; give GitHub a moment to serve the repo.
    for _ in $(seq 1 30); do
      gh repo view "${FORK_OWNER}/${name}" >/dev/null 2>&1 && break
      sleep 2
    done
  fi
  gh repo view "${FORK_OWNER}/${name}" >/dev/null 2>&1 || fail "fork ${FORK_OWNER}/${name} is not available"
}

prepare_checkout() {
  local upstream=$1
  local dir=$2
  local branch=$3
  local name=${upstream#*/}

  if [[ ! -d ${dir}/.git ]]; then
    log "cloning ${FORK_OWNER}/${name}"
    git clone --depth 1 "https://github.com/${FORK_OWNER}/${name}.git" "${dir}"
  fi

  git -C "${dir}" fetch --depth 1 "https://github.com/${upstream}.git" main
  git -C "${dir}" checkout -q -B "${branch}" FETCH_HEAD
  git -C "${dir}" config user.name "${GIT_NAME}"
  git -C "${dir}" config user.email "${GIT_EMAIL}"
}

copy_bundle() {
  local dir=$1
  local operator_dir="${dir}/operators/kubernetes-imagepuller-operator/${VERSION}"
  local csv="${operator_dir}/manifests/kubernetes-imagepuller-operator.v${VERSION}.clusterserviceversion.yaml"

  if [[ -f ${csv} ]]; then
    log "bundle ${VERSION} is already committed in ${dir}"
  else
    rm -rf "${operator_dir}"
    (cd "${REPO_ROOT}" && ./make-release.sh "${VERSION}" \
      --prepare-community-operators-update --community-operators-repository-dir "${dir}")
  fi

  [[ -f ${csv} ]] || fail "${csv} was not created"
  local csv_version
  csv_version="$(${YQ_PATH} -r '.spec.version' "${csv}")"
  [[ ${csv_version} == "${VERSION}" ]] || \
    fail "the bundle in ${dir} declares version '${csv_version}', not '${VERSION}'"
}

commit_bundle() {
  local dir=$1
  local subject=$2

  if [[ -z $(git -C "${dir}" status --porcelain) ]]; then
    log "nothing to commit in ${dir}"
    return
  fi

  git -C "${dir}" add "operators/kubernetes-imagepuller-operator/${VERSION}"
  git -C "${dir}" commit -s \
    -m "${subject}" \
    -m "Bundle for kubernetes-image-puller-operator ${VERSION} (operator image quay.io/eclipse/kubernetes-image-puller-operator:${VERSION}), replacing the previous version." >/dev/null
  log "committed $(git -C "${dir}" log -1 --format='%h %s')"
}

pr_body() {
  cat <<EOF
Updates kubernetes-imagepuller-operator to ${VERSION}.

- Operator image: \`quay.io/eclipse/kubernetes-image-puller-operator:${VERSION}\`
- Bundle image: \`quay.io/eclipse/kubernetes-image-puller-operator-bundle:${VERSION}\`
- CSV \`kubernetes-imagepuller-operator.v${VERSION}\` with \`replaces\` pointing at the previous release (\`replaces-mode\` via \`ci.yaml\`)

Verification performed before opening this pull request:

- [x] \`operator-sdk bundle validate --select-optional suite=operatorframework\` passes for this bundle
- [x] Operator deployed on Kubernetes, \`KubernetesImagePuller\` reached \`Ready\`
- [x] Image puller DaemonSet scheduled and pods ready
- [x] CR update propagated and CR deletion cleaned up owned resources
- [x] Validating webhook rejects a second \`KubernetesImagePuller\` in the same namespace
- [x] Operator installed from the published bundle with OLM (\`CSV Succeeded\`)
- [x] Commit is signed off (DCO)

Release branch: https://github.com/che-incubator/kubernetes-image-puller-operator/tree/${VERSION}-release
Release tag: https://github.com/che-incubator/kubernetes-image-puller-operator/releases/tag/${VERSION}
EOF
}

publish() {
  local upstream=$1
  local subject=$2
  local name=${upstream#*/}
  local dir="${WORK_DIR}/${name}"
  local branch="kip-${VERSION}"

  ensure_fork "${upstream}"
  prepare_checkout "${upstream}" "${dir}" "${branch}"

  # Capture the tip of the fork branch before preparing the replacement commit.
  # Reading it afterwards would adopt a concurrent update as the expected lease
  # and then silently overwrite it. A shallow clone has no tracking ref for the
  # branch, so ask the remote instead of relying on one.
  local lease
  lease="$(git -C "${dir}" ls-remote --heads origin "refs/heads/${branch}" | cut -f1)"

  copy_bundle "${dir}"
  commit_bundle "${dir}" "${subject}"

  if [[ ${DRY_RUN} == "true" ]]; then
    log "dry run: showing changes for ${upstream}"
    git -C "${dir}" show --stat --oneline HEAD | head -20
    log "dry run: skipping push and pull request for ${upstream}"
    return
  fi

  # Lease against the tip observed before preparation so that reruns replace only
  # what this workflow published earlier.
  if [[ -n ${lease} ]]; then
    git -C "${dir}" push --force-with-lease="refs/heads/${branch}:${lease}" -u origin "${branch}" >/dev/null
  else
    git -C "${dir}" push -u origin "${branch}" >/dev/null
  fi
  log "pushed ${FORK_OWNER}:${branch}"

  local body
  body="$(mktemp)"
  pr_body >"${body}"

  # Pull requests from a fork are only matched by their qualified head ref.
  local selector="${FORK_OWNER}:${branch}"
  local number=""
  number="$(gh pr view "${selector}" --repo "${upstream}" --json number --jq .number 2>/dev/null)" || true
  if [[ -z ${number} ]]; then
    log "opening pull request against ${upstream}"
    gh pr create --repo "${upstream}" --base main --head "${selector}" \
      --title "${subject}" --body-file "${body}"
    rm -f "${body}"
    return
  fi

  # A closed or merged pull request is still returned by the lookup above, but
  # publishing has to leave an open one behind.
  local state
  state="$(gh pr view "${number}" --repo "${upstream}" --json state --jq .state)"
  case ${state} in
    OPEN) ;;
    CLOSED)
      log "reopening the closed pull request #${number} in ${upstream}"
      gh pr reopen "${number}" --repo "${upstream}" >/dev/null
      ;;
    MERGED)
      log "pull request #${number} in ${upstream} is already merged; nothing to publish"
      rm -f "${body}"
      return
      ;;
    *)
      fail "pull request #${number} in ${upstream} has unexpected state '${state}'"
      ;;
  esac

  log "updating pull request #${number} in ${upstream}"
  gh pr edit "${number}" --repo "${upstream}" --title "${subject}" --body-file "${body}" >/dev/null
  rm -f "${body}"
}

main() {
  parse_args "$@"
  preflight
  publish "${UPSTREAM_COMMUNITY_OPERATORS}" "operator kubernetes-imagepuller-operator (${VERSION})"
  publish "${UPSTREAM_COMMUNITY_OPERATORS_PROD}" "chore: KIP ${VERSION}"
  log "done"
}

main "$@"
