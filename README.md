# Kubernetes Image Puller Operator

[![Contribute](https://che.openshift.io/factory/resources/factory-contribute.svg)](https://che.openshift.io/f?url=https://github.com/che-incubator/kubernetes-image-puller-operator)

An operator to install, configure, and manage a [kubernetes-image-puller](https://github.com/che-incubator/kubernetes-image-puller) deployment.

Docs to come


### Releasing a new OLM bundle

After adding a new version directory under `deploy/olm-catalog/kubernetes-image-puller-operator`, and new `ClusterServiceVersion`s and CRDs:

1. Make a branch with name `olm-vX.Y.Z`
2. Change the version in `version/version.go` to match your new operator bundle version.  If your new bundle is v0.5.1, for example, this is what the `Version` var should be in `version/version.go`.
3. Run `operator-courier verify --ui_validate_io deploy/olm-catalog/kubernetes-image-puller-operator` locally to make sure everything works OK.
4. Open a PR to master.  When this PR is merged and closed, a github action will push the new version of the operator bundle to quay.io/eclipse
