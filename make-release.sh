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

init() {
  RELEASE_VERSION="$1"
  RELEASE_IMAGE="$(make base-image):${RELEASE_VERSION}"
  RELEASE_BRANCH="${RELEASE_VERSION}-release"
  RUN_RELEASE=false
  RELEASE_OLM_FILES=false
  RELEASE_OLM_BUNDLE=false
  PUSH_GIT_CHANGES=false
  FORCE_UPDATE=""
  CREATE_PULL_REQUESTS=false
  GIT_REMOTE_UPSTREAM="https://github.com/che-incubator/kubernetes-image-puller-operator.git"
  PREPARE_COMMUNITY_OPERATORS_UPDATES=false
  COMMUNITY_OPERATORS_REPOSITORY_DIR=""

  if [[ $# -lt 1 ]]; then usage; exit; fi

  while [[ "$#" -gt 0 ]]; do
    case $1 in
      '--release') RUN_RELEASE=true; shift 0;;
      '--release-olm-files') RELEASE_OLM_FILES=true; shift 0;;
      '--release-olm-bundle') RELEASE_OLM_BUNDLE=true; shift 0;;
      '--push-git-changes') PUSH_GIT_CHANGES=true; shift 0;;
      '--force') FORCE_UPDATE="--force"; shift 0;;
      '--pull-requests') CREATE_PULL_REQUESTS=true; shift 0;;
      '--prepare-community-operators-update') PREPARE_COMMUNITY_OPERATORS_UPDATES=true; shift 0;;
      '--community-operators-repository-dir') COMMUNITY_OPERATORS_REPOSITORY_DIR=$2; shift 1;;
    '--help'|'-h') usage; exit;;
    esac
    shift 1
  done
}

usage () {
  echo "Usage:   $0 [RELEASE_VERSION] --release --release-olm-files --release-olm-bundle --push-git-changes --pull-requests"
  echo "Usage:   $0 [RELEASE_VERSION] --prepare-community-operators-update --community-operators-repository-dir <DIR>"
}

resetChanges() {
  echo "[INFO] Reset changes in $1 branch"
  git reset --hard
  git checkout $1
  git fetch ${GIT_REMOTE_UPSTREAM} --prune
  git pull ${GIT_REMOTE_UPSTREAM} $1
}

runUnitTests() {
  echo "[INFO] runUnitTests :: Run unit tests"
  make test
}

checkoutToReleaseBranch() {
  echo "[INFO] Check out to $RELEASE_BRANCH branch."
  resetChanges main
  git push origin main:"$RELEASE_BRANCH" --force
  git checkout -B "$RELEASE_BRANCH"
}

buildOperatorImage() {
  echo "[INFO] buildOperatorImage :: Build operator image"
  make docker-build docker-push IMG="${RELEASE_IMAGE}"
}

updateVersionFile() {
  echo "[INFO] updateVersionFile: version.go"
  # change version/version.go file
  sed -ri "s/Version = \"[0-9]+.[0-9]+.[0-9]\"/Version = \"${RELEASE_VERSION}\"/g" version/version.go
  git add version/version.go

  # Set up new version for Makefile and version.go.
  echo "[INFO] updateVersionFile: Makefile"
  sed -ri "s/VERSION \?= [0-9]+.[0-9]+.[0-9]/VERSION \?= ${RELEASE_VERSION}/g" Makefile
  git add Makefile

  git commit -m "ci: Update VERSION to ${RELEASE_VERSION}" --signoff
}

releaseOlmFiles() {
  echo "[INFO] releaseOlmFiles"

  echo "[INFO] releaseOlmFiles :: Release OLM files"

  CURRENT_VERSION=$(make bundle-version)
  PACKAGE=$(make bundle-package)
  CSV_PATH=$(make csv-path)
  CREATE_TIMESTAMP=$(date +"%Y-%m-%dT%H:%M:%SZ")

  make bundle IMG="${RELEASE_IMAGE}"

  yq -riY '.metadata.name = "'${PACKAGE}'.v'${RELEASE_VERSION}'"' ${CSV_PATH}
  yq -riY '.metadata.annotations.createdAt = "'${CREATE_TIMESTAMP}'"' ${CSV_PATH}
  yq -riY '.spec.version = "'${RELEASE_VERSION}'"' ${CSV_PATH}
  yq -riY '.spec.replaces = "'${PACKAGE}'.v'${CURRENT_VERSION}'"' ${CSV_PATH}

  make license "$(make bundle-path)"

  echo "[INFO] releaseOlmFiles :: Commit changes"
  if git status --porcelain; then
    git add -A || true
    git commit -am "ci: Release OLM files to "${RELEASE_VERSION} --signoff
  fi
}

releaseOlmBundle() {
  echo "[INFO] releaseOlmBundle"

  CHANNEL_PATH=$(make channel-path)
  BUNDLE_NAME=$(make bundle-name)
  CATALOG_IMAGE=$(make catalog-image)
  BUNDLE_IMAGE=$(make bundle-image)

  echo "[INFO] Bundle name   : ${BUNDLE_NAME}"
  echo "[INFO] Catalog image : ${CATALOG_IMAGE}"
  echo "[INFO] Bundle image  : ${BUNDLE_IMAGE}"

  if [[ $(yq -r '.entries[] | select(.name == "'${BUNDLE_NAME}'")' "${CHANNEL_PATH}") ]]; then
    echo "[ERROR] Bundle ${BUNDLE_NAME} already exists in the catalog"
    exit 1
  else
    echo "[INFO] releaseOlmBundle :: Build and push the new bundle image to quay.io..."
    make bundle-build bundle-push

    echo "[INFO] releaseOlmBundle :: Add bundle to the catalog..."
    LAST_BUNDLE_NAME=$(yq -r '.entries | .[length - 1].name' "${CHANNEL_PATH}")
    make bundle-render
    yq -riY '(.entries) += [{"name": "'${BUNDLE_NAME}'", "replaces": "'${LAST_BUNDLE_NAME}'"}]' "${CHANNEL_PATH}"
  fi

  echo "[INFO] releaseOlmBundle :: Build and push the catalog image to quay.io..."
  make catalog-build catalog-push

  echo "[INFO] releaseOlmBundle :: Commit changes"
  make license "$(make catalog-path)"
  git add -A olm-catalog/stable
  git commit -m "ci: Add new bundle to a catalog" --signoff
}

pushGitChanges() {
  echo "[INFO] Push git changes into $RELEASE_BRANCH branch"
  git push origin $RELEASE_BRANCH ${FORCE_UPDATE}
  if [[ $FORCE_UPDATE == "--force" ]]; then # if forced update, delete existing tag so we can replace it
    if [[ $(git tag -l ${RELEASE_VERSION}) ]]; then # if tag exists in local repo
      echo "Remove existing local tag ${RELEASE_VERSION}"
      git tag -d ${RELEASE_VERSION}
    else
      echo "Local tag ${RELEASE_VERSION} does not exist" # should never get here
    fi
    if [[ $(git ls-remote --tags $(git remote get-url origin) ${RELEASE_VERSION}) ]]; then # if tag exists in remote repo
      echo "Remove existing remote tag ${RELEASE_VERSION}"
      git push origin :${RELEASE_VERSION}
    else
      echo "Remote tag ${RELEASE_VERSION} does not exist" # should never get here
    fi
  fi
  git tag -a ${RELEASE_VERSION} -m ${RELEASE_VERSION}
  git push --tags origin
}

createPRToMainBranch() {
  echo "[INFO] createPRToMainBranch :: Create pull request into main branch"
  resetChanges main
  local tmpBranch="copy-${RELEASE_VERSION}-bundle-to-main"
  git checkout -B "${tmpBranch}"
  git diff refs/heads/main...refs/heads/${RELEASE_BRANCH} | git apply -3
  if git status --porcelain; then
    git add -A || true
    git commit -am "ci: Copy ${RELEASE_VERSION} bundle to main" --signoff
  fi
  git push origin $tmpBranch -f
  if [[ $FORCE_UPDATE == "--force" ]]; then set +e; fi  # don't fail if PR already exists (just force push commits into it)
  gh pr create -f -B main -H ${tmpBranch} -t "ci: Copy ${RELEASE_VERSION} bundle to main"
  set -e
}

prepareCommunityOperatorsUpdates() {
  echo "[INFO] Copy bundle into community-operators repository"
  if [[ ! ${RELEASE_VERSION} = $(make bundle-version) ]]; then
    echo "[ERROR] Release version ${RELEASE_VERSION} does not match bundle version $(make bundle-version)"
    exit 1
  fi

  BUNDLE_DIR=$(make bundle-path)
  KIP_DIR="${COMMUNITY_OPERATORS_REPOSITORY_DIR}/operators/kubernetes-imagepuller-operator/$(make bundle-version)"

  mkdir "$KIP_DIR"
  cp -r "${BUNDLE_DIR}/manifests" "${KIP_DIR}"
  cp -r "${BUNDLE_DIR}/metadata" "${KIP_DIR}"
  cp -r "${BUNDLE_DIR}/tests" "${KIP_DIR}"

  mv "${KIP_DIR}/manifests/$(make bundle-package).clusterserviceversion.yaml" \
     "${KIP_DIR}/manifests/$(make bundle-package).v$(make bundle-version).clusterserviceversion.yaml"
}

run() {
  runUnitTests
  checkoutToReleaseBranch
  buildOperatorImage
  if [[ $RELEASE_OLM_FILES == "true" ]]; then
    releaseOlmFiles
    updateVersionFile
  fi
}

init "$@"
echo "[INFO] Release: $RELEASE_VERSION"

if [[ $RUN_RELEASE == "true" ]]; then
  run "$@"
fi

if [[ $RELEASE_OLM_BUNDLE == "true" ]]; then
  releaseOlmBundle
fi

if [[ $PUSH_GIT_CHANGES == "true" ]]; then
  pushGitChanges
fi

if [[ $CREATE_PULL_REQUESTS == "true" ]]; then
  createPRToMainBranch
fi

if [[ $PREPARE_COMMUNITY_OPERATORS_UPDATES == "true" ]]; then
  prepareCommunityOperatorsUpdates
fi