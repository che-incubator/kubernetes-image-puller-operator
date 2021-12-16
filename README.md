![Next Build Status](https://github.com/che-incubator/kubernetes-image-puller-operator/actions/workflows/next-build.yml/badge.svg)
![Release Build Status](https://github.com/che-incubator/kubernetes-image-puller-operator/actions/workflows/release.yml/badge.svg)

# Kubernetes Image Puller Operator

[![Contribute](https://www.eclipse.org/che/contribute.svg)](https://workspaces.openshift.com#https://github.com/che-incubator/kubernetes-image-puller-operator)

An operator to install, configure, and manage a [kubernetes-image-puller](https://github.com/che-incubator/kubernetes-image-puller) deployment.

The kubernetes-image-puller creates daemonsets that will run a list of images on a cluster, allowing Eclipse Che to start workspaces faster, because those images have been pre-pulled.  For more information about the kubernetes-image-puller, consult the kubernetes-image-puller [README](https://github.com/che-incubator/kubernetes-image-puller/blob/master/README.md).

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
  	che-theia=quay.io/eclipse/che-theia:next;java11-maven=quay.io/eclipse/che-java11-maven:next
  cachingIntervalHours: '2' # number of hours between health checks
  cachingMemoryRequest: '10Mi' # the memory request for each pre-pulled image
  cachingMemoryLimit: '20Mi' # the memory limit for each pre-pulled image
  nodeSelector: '{}' # node selector applied to pods created by the daemonset
```

### Installing The Operator

#### Installing from OperatorHub

> Notice: to install operator using OperatorHub you need to have Kubernetes cluster with pre-installed OLM. 
You can install OLM on the cluster using operator-sdk "operator-sdk olm install". OLM is pre-installed On the Openshift clusters since version 4.2.

Open OperatorHub page https://operatorhub.io/operator/kubernetes-imagepuller-operator. Click install button and follow instructions.
When operator pod will be ready click on link "View YAML Example" to get custom resource yaml(CR) in the bottom of the page.
Store this yaml into file, edit with desired operator configuration and apply using:

```bash
$ kubect apply -f <CR-YAML-FILE>.yaml -n <IMAGE-PULLER-NAMESPACE>
```

#### Installing Manually

Clone this project, checkout to desired release tag.

For version <= v0.0.9 use installation commands:

```bash
kubectl apply -f deploy/ -n <NAMESPACE>
kubectl apply -f deploy/crds/ -n <NAMESPACE>
```

For version >= v0.0.10 type in the terminal:

``` bash
$ make deploy -s
```

Customize custom resource yaml and apply it:

```bash
$ kubectl apply -f config/samples/che_v1alpha1_kubernetesimagepuller.yaml -n kubernetes-image-puller-operator
```

To uninstall operator with version <= v0.0.9 use commands:

```bash
kubectl delete -f deploy/ -n <NAMESPACE>
kubectl delete -f deploy/crds/ -n <NAMESPACE>
```

To uninstall operator with version >= 0.0.10 use command:

```bash
make undeploy -s
```

### Set up Prometeus from scratch and provide metrics

Install kubernetes-image-puller operator. 

Install Kube Prometheus https://github.com/prometheus-operator/kube-prometheus#quickstart

Await while all pods in the `metrics` namespace are running. 

You can expose Prometheus dashboard: https://github.com/prometheus-operator/kube-prometheus#access-the-dashboards or use ingresses for this purpose.

Provide more permission for Prometheuse to watch metrics in the another namespaces:

```bash
kubectl patch clusterrole prometheus-k8s --type='json' -p='[{"op": "add", "path": "/rules/0", "value":
{"apiGroups": [""], "resources": ["services", "endpoints", "pods"], "verbs": ["get","list","watch"]}}]'

kubectl patch clusterrole prometheus-k8s --type='json' -p='[{"op": "add", "path": "/rules/0", "value":
{"apiGroups": ["extensions"], "resources": ["ingresses"], "verbs": ["get","list","watch"]}}]'

kubectl patch clusterrole prometheus-k8s --type='json' -p='[{"op": "add", "path": "/rules/0", "value":
{"apiGroups": ["networking.k8s.io"], "resources": ["ingresses"], "verbs": ["get","list","watch"]}}]'
```

Configure Prometeus to find kubernetes-image-puller operator metrics by labels:

```bash
kubectl patch prometheus k8s -n monitoring --type='json' -p '[{ "op": "add",
"path": "/spec/serviceMonitorSelector", "value": {
    "matchExpressions": [
      {
        "key": "name",
        "operator": "In",
        "values": [
          "kubernetes-image-puller-operator"
        ]
      }
    ]
  }
}]'

```
    serviceMonitorSelector:
      matchExpressions:
      - key: name
        operator: In
        values:
        - kubernetes-image-puller-operator

Apply kubernetes-image-puller ServiceMonitor:

```
kubectl apply -f config/prometheus/monitor.yaml
```

### Development

#### Prequisites
* Go >=`1.15`
* Operator SDK `v1.7.1` (recommended)

To build custom development images set up env variables:

```bash
$ export IMAGE_REGISTRY_USER_NAME=<IMAGE_REGISTRY_USER_NAME> && \
  export IMAGE_REGISTRY_HOST=<IMAGE_REGISTRY_HOST>
```

Where:
- `IMAGE_REGISTRY_USER_NAME` - docker image registry account name.
- `IMAGE_REGISTRY_HOST` - docker image registry hostname, for example: "docker.io", "quay.io".

> Warning: if you are using quay.io, for all new images you need to go to the image web page and make image publicity,
otherwise you will face with error pull issue.

> Notice: you can store this env variables into the ${HOME}/.bashrc file.

#### Check code compilation

Run VSCode task `Compile code` or use the terminal:

```bash
$ make compile -s
```

#### Unit testing

Run VSCode task `Run unit tests` or use the terminal:

```bash
$ make test
```

#### Format code

Run VSCode task `Format code` or use the terminal:

```bash
$ go fmt ./...
```

#### Update golang dependencies

This project uses Go modules and doesn't use vendor folder. Run the VSCode task: `Update dependencies` or use the terminal:

```bash
$ go mod tidy
```

#### Building custom operator image

Run VSCode task `Build and push custom operator image: '${IMAGE_REGISTRY_HOST}/${IMAGE_REGISTRY_USER_NAME}/kubernetes-image-puller-operator:next'` or use the terminal:

```bash
$ make docker-build docker-push IMG="${IMAGE_REGISTRY_HOST}/${IMAGE_REGISTRY_USER_NAME}/kubernetes-image-puller-operator:next"
```

#### Installing using make

Build and push custom operator image if you modified source code.
Run VSCode task `Deploy operator` or use the terminal:

```bash
$ make deploy IMG="${IMAGE_REGISTRY_HOST}/${IMAGE_REGISTRY_USER_NAME}/kubernetes-image-puller-operator:next" -s
$ kubectl apply -f config/samples/che_v1alpha1_kubernetesimagepuller.yaml -n <NAMESPACE>
```

To uninstall operator run VSCode task `Undeploy operator`

```bash
$ make undeploy -s
```

#### Installing using operator-sdk and OLM

Build and push custom operator image if you modified source code. 
Build new OLM bundle image using VSCode task `Build and push development bundle` or use the terminal:

```bash
$ export BUNDLE_IMG="${IMAGE_REGISTRY_HOST}/${IMAGE_REGISTRY_USER_NAME}/kubernetes-image-puller-operator-bundle:next"
$ make bundle IMG=${BUNDLE_IMG} -s
$ make bundle-build bundle-push -s BUNDLE_IMG=${BUNDLE_IMG}
```

To install operator run VSCode task `Install operator via OLM` or use the terminal:

```bash
$ operator-sdk run bundle ${IMAGE_REGISTRY_HOST}/${IMAGE_REGISTRY_USER_NAME}/kubernetes-image-puller-operator-bundle:next --namespace <NAMESPACE>
$ kubectl apply -f config/samples/che_v1alpha1_kubernetesimagepuller.yaml -n <NAMESPACE>
```

To uninstall operator run VSCode task `UnInstall operator via OLM` or use the terminal:

```bash
$ operator-sdk cleanup kubernetes-imagepuller-operator --namespace <NAMESPACE>
```

#### Update CR/CRD

Run VSCode task `Update CR/CRDs` or use the terminal:

```bash
$ make generate manifests -s
```

#### Update OLM bundle

Run VSCode task `Update OLM bundle` or use the terminal:

```bash
$ make bundle -s
```

#### Releasing a new version of the operator to OperatorHub

A quirk of this project is that while the repository is named `kubernetes-image-puller-operator`, the operator bundle on OperatorHub is named `kubernetes-imagepuller-operator`.  This was caused by the previous version of OLM deployment that required a Quay.io Application.  

Make release bundle:

```bash
$ make release-bundle -s RELEASE_VERSION=0.0.10
```

This command will convert OLM bundle(from `bundle` folder) to the package manifest with newer release
version and put it to the directory `deploy/olm-catalog/kubernetes-imagepuller-operator/VERSION`.
You now have a new version of the CSV and CRD, and can open a PR to `kubernetes-image-puller-operator` to keep everything in sync.

Then, to see these changes on OperatorHub:
1. Clone the [`community-operators`](https://github.com/k8s-operatorhub/community-operators) and [`community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/) repositories.
2. Copy `deploy/olm-catalog/kubernetes-imagepuller-operator/` to the `kubernetes-imagepuller-operator` folder of those repositories.
3. Optionally run some tests to confirm your changes are good (see [testing operators](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md))
4. Open two separate pull requests to [`community-operators`](https://github.com/k8s-operatorhub/community-operators/) and [`community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/) repositories. Examples of the PRs:
- https://github.com/k8s-operatorhub/community-operators/pull/96
- https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/87
