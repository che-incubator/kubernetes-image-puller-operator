# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 1.0.4

# Add silent flag for all commands by default
ifndef VERBOSE
	MAKEFLAGS += --silent
endif

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
CHECLUSTER_CRD_PATH = "$(PROJECT_DIR)/config/crd/bases/che.eclipse.org_kubernetesimagepullers.yaml"

# CHANNEL define the bundle package name
PACKAGE = kubernetes-imagepuller-operator

# CHANNEL define the bundle channel
CHANNEL = stable

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
IMAGE_TAG_BASE ?= quay.io/eclipse/kubernetes-image-puller-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):next

# The image tag given to the resulting catalog image
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:$(CHANNEL)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Detect image tool
ifeq ($(IMAGE_TOOL),)
ifneq (,$(shell which docker))
	IMAGE_TOOL := docker
else
	IMAGE_TOOL := podman
endif
endif

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "[INFO] Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef


# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
# SHELL = /usr/bin/env bash -o pipefail
# .SHELLFLAGS = -ec
.ONESHELL:

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

##@ Development

manifests: download-controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd:crdVersions=v1 rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	#$(CONTROLLER_GEN) crd:trivialVersions=true,preserveUnknownFields=false rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

	# remove yaml delimitier, which makes OLM catalog source image broken.
	sed -i '/---/d' "$(CHECLUSTER_CRD_PATH)"

	$(MAKE) license $$(find ./config/crd -not -path "./vendor/*" -name "*.yaml")

generate: download-controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

install: manifests download-kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests download-kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests download-kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd "$(PROJECT_DIR)/config/manager"
	$(KUSTOMIZE) edit set image quay.io/eclipse/kubernetes-image-puller-operator:next=${IMG}
	cd "$(PROJECT_DIR)"

	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: download-kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

compile:
	binary="$(BINARY)"
	if [ -z "$${binary}" ]; then
		binary="/tmp/image-puller/kubernetes-image-puller-operator"
	fi
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build -a -o "$${binary}" main.go
	echo "kubernetes-image-puller-operator binary compiled to $${binary}"

##@ OLM catalog

.PHONY: bundle
bundle: generate manifests download-kustomize download-operator-sdk ## Generate OLM bundle
	echo "[INFO] Updating OperatorHub bundle"

	# Build default clusterserviceversion file
	$(OPERATOR_SDK) generate kustomize manifests

	BUNDLE_PATH=$$($(MAKE) bundle-path)

	$(KUSTOMIZE) build config/manifests | \
	$(OPERATOR_SDK) generate bundle \
	--quiet \
	--overwrite \
	--version $(VERSION) \
	--package $(PACKAGE) \
	--output-dir $${BUNDLE_PATH} \
	--channels $(CHANNEL) \
	--default-channel $(CHANNEL)

	# Copy bundle.Dockerfile to the bundle dir
 	# Update paths (since it is created in the root of the project) and labels
	mv bundle.Dockerfile $${BUNDLE_PATH}
	sed -i 's|$(PROJECT_DIR)/bundle/||' $${BUNDLE_PATH}/bundle.Dockerfile

	make license $${BUNDLE_PATH}

	$(OPERATOR_SDK) bundle validate $${BUNDLE_PATH}

bundle-build: download-opm ## Build a bundle image
	BUNDLE_PATH=$$($(MAKE) bundle-path)
	$(IMAGE_TOOL) build -f $${BUNDLE_PATH}/bundle.Dockerfile -t $(BUNDLE_IMG) $${BUNDLE_PATH}

bundle-push: ## Push a bundle image
	$(IMAGE_TOOL) push $(BUNDLE_IMG)

bundle-render: SHELL := /bin/bash
bundle-render: download-opm ## Add bundle to a catalog
	CATALOG_PATH=$$($(MAKE) catalog-path)
	BUNDLE_NAME=$$($(MAKE) bundle-name)

	$(OPM) render $(BUNDLE_IMG) -o yaml --skip-tls-verify | sed 's|---||g' > $${CATALOG_PATH}/$${BUNDLE_NAME}.bundle.yaml

bundle-path: ## Prints path to a bundle
	echo "$(PROJECT_DIR)/bundle"

base-image: ## Prints operator image name
	echo $(IMAGE_TAG_BASE)

bundle-image: ## Prints bundle image name
	echo $(BUNDLE_IMG)

catalog-path: ## Prints path to a catalog directory
	echo "$(PROJECT_DIR)/olm-catalog/stable"

catalog-image: ## Prints catalog image name
	echo $(CATALOG_IMG)

channel-path: ## Prints path to a channel.yaml
	CATALOG_PATH=$$($(MAKE) catalog-path)
	echo "$${CATALOG_PATH}/channel.yaml"

csv-path: ## Prints path to a clusterserviceversion file
	BUNDLE_PATH=$$($(MAKE) bundle-path)
	echo "$${BUNDLE_PATH}/manifests/$(PACKAGE).clusterserviceversion.yaml"

bundle-package: ## Prints a package name
	echo $(PACKAGE)

bundle-name: ## Prints a clusterserviceversion
	CSV_PATH=$$($(MAKE) csv-path)
	echo $$(yq -r ".metadata.name" "$${CSV_PATH}")

bundle-version: ## Prints a bundle version
	CSV_PATH=$$($(MAKE) csv-path)
	echo $$(yq -r ".spec.version" "$${CSV_PATH}")

catalog-build: download-opm ## Build a catalog image
	CATALOG_PATH=$$($(MAKE) catalog-path)

	$(OPM) validate $${CATALOG_PATH}
	$(IMAGE_TOOL) build -f $${CATALOG_PATH}/../index.Dockerfile -t $(CATALOG_IMG) .

catalog-push: ## Push a catalog image
	$(IMAGE_TOOL) push $(CATALOG_IMG)

##@ Utilities

OPM ?= $(shell pwd)/bin/opm
OPM_VERSION = v1.26.2
download-opm: SHELL := /bin/bash
download-opm: ## Download opm tool
	[[ -z "$(DEST)" ]] && dest=$(OPM) || dest=$(DEST)/opm
	command -v $(OPM) >/dev/null 2>&1 && exit

	OS=$(shell go env GOOS)
	ARCH=$(shell go env GOARCH)

	echo "[INFO] Downloading opm version: $(OPM_VERSION)"

	mkdir -p $$(dirname "$${dest}")
	curl -sL https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$${OS}-$${ARCH}-opm > $${dest}
	chmod +x $${dest}


OPERATOR_SDK ?= $(shell pwd)/bin/operator-sdk
OPERATOR_SDK_VERSION = v1.9.2
download-operator-sdk: SHELL := /bin/bash
download-operator-sdk: ## Downloads operator sdk tool
	[[ -z "$(DEST)" ]] && dest=$(OPERATOR_SDK) || dest=$(DEST)/operator-sdk
	command -v $${dest} >/dev/null 2>&1 && exit

	OS=$(shell go env GOOS)
	ARCH=$(shell go env GOARCH)

	echo "[INFO] Downloading operator-sdk version $(OPERATOR_SDK_VERSION) into $${dest}"
	mkdir -p $$(dirname "$${dest}")
	curl -sL https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} > $${dest}
	chmod +x $${dest}

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
download-controller-gen: ## Download controller-gen tool
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
download-kustomize: ## Download kustomize tool
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.7)


ADD_LICENSE = $(shell pwd)/bin/addlicense
download-addlicense: ## Download addlicense tool
	$(call go-get-tool,$(ADD_LICENSE),github.com/google/addlicense@99ebc9c9db7bceb8623073e894533b978d7b7c8a)

license: download-addlicense ## Add license to the files
	FILES=$$(echo $(filter-out $@,$(MAKECMDGOALS)))
	$(ADD_LICENSE) -f hack/license-header.txt $${FILES}