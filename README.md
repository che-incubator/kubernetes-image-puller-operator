[![master](https://ci.centos.org/buildStatus/icon?subject=master&job=devtools-che-incubator-kubernetes-image-puller-operator-build-master/)](https://ci.centos.org/job/devtools-che-incubator-kubernetes-image-puller-operator-build-master/)
[![nightly](https://ci.centos.org/buildStatus/icon?subject=nightly&job=devtools-kubernetes-image-puller-operator-nightly)](https://ci.centos.org/job/devtools-kubernetes-image-puller-operator-nightly/)


# Kubernetes Image Puller Operator

[![Contribute](https://che.openshift.io/factory/resources/factory-contribute.svg)](https://che.openshift.io/f?url=https://github.com/che-incubator/kubernetes-image-puller-operator)

An operator to install, configure, and manage a [kubernetes-image-puller](https://github.com/che-incubator/kubernetes-image-puller) deployment.

The kubernetes-image-puller creates daemonsets that will run a list of images on your cluster, allowing Eclipse Che to start workspaces faster, because those images have been pre-pulled.  For more information about the kubernetes-image-puller, consult the kubernetes-image-puller [README](https://github.com/che-incubator/kubernetes-image-puller/blob/master/README.md).

The operator provides a `KubernetesImagePuller` custom resource definition (CRD) to install and configure a kubernetes-image-puller instance.

### Example Custom Resource

```yaml
apiVersion: che.eclipse.org/v1alpha1
kind: KubernetesImagePuller
metadata:
  name: image-puller
spec:
  deploymentName: kubernetes-image-puller # the name of the deployment the operator creates
  configMapName: k8s-image-puller # the name of the configmap the operator creates
  daemonsetName: k8s-image-puller # the name of subsequent daemonsets created by the kubernetes-image-puller
  images: >- # the list of images to pre-pull
  	che-theia=quay.io/eclipse/che-theia:next;java11-maven=quay.io/eclipse/che-java11-maven:nightly
  cachingIntervalHours: '2' # number of hours between health checks
  cachingMemoryRequest: '10Mi' # the memory request for each pre-pulled image
  cachingMemoryLimit: '20Mi' # the memory limit for each pre-pulled image
  nodeSelector: '{}' # node selector applied to pods created by the daemonset
```

### Installing The Operator

#### Installing on OperatorHub

#### Installing Manually
``` shell
kubectl apply -f deploy/
kubectl apply -f deploy/crds/
```

There are pull requests to add the operator to the community-operators repository so you can install via OLM:

https://github.com/operator-framework/community-operators/pull/1661

https://github.com/operator-framework/community-operators/pull/1662

### Building

`operator-sdk build quay.io/eclipse/kubernetes-image-puller-operator:tag`

### Testing

`go test -v ./pkg... ./cmd...`

### Releasing a new OLM bundle

After adding a new version directory under `deploy/olm-catalog/kubernetes-image-puller-operator`, and new `ClusterServiceVersion`s and CRDs:

1. Make a branch with name `olm-vX.Y.Z`
2. Change the version in `version/version.go` to match your new operator bundle version.  If your new bundle is v0.5.1, for example, this is what the `Version` var should be in `version/version.go`.
3. Run `operator-courier verify --ui_validate_io deploy/olm-catalog/kubernetes-image-puller-operator` locally to make sure everything works OK.
4. Open a PR to master.  When this PR is merged and closed, a github action will push the new version of the operator bundle to quay.io/eclipse
