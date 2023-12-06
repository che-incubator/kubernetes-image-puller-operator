ifeq (,$(shell which kubectl)$(shell which oc))
    $(error oc or kubectl is required to proceed)
endif

ifneq (,$(shell which kubectl))
    K8S_CLI := kubectl
else
    K8S_CLI := oc
endif

ifeq ($(shell $(K8S_CLI) api-resources --api-group='route.openshift.io' 2>&1 | grep -o routes),routes)
    PLATFORM := openshift
else
    PLATFORM := kubernetes
endif

install: manifests download-kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(K8S_CLI) apply -f -

uninstall: download-kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(K8S_CLI) delete -f -

deploy: manifests kustomize-operator-image gen-deployment ## Deploy controller to the K8s cluster specified in ~/.kube/config.	
	$(K8S_CLI) apply -f $(DEPLOYMENT_DIR)/$(PLATFORM)/combined.yaml

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(K8S_CLI) delete -f $(DEPLOYMENT_DIR)/$(PLATFORM)/combined.yaml
