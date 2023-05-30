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

Clone this project, checkout to desired release tag and type in the terminal:

```bash
$ make deploy
```

Customize custom resource yaml and apply it:

```bash
$ kubectl apply -f config/samples/che_v1alpha1_kubernetesimagepuller.yaml -n kubernetes-image-puller-operator
```

To uninstall operator with version <= v0.0.9 use commands:

```bash
$ make undeploy -s
```

### Set up Prometheus from scratch on the Kubernetes cluster and provide metrics

Install kubernetes-image-puller operator on the Kubernetes cluster(for example Minikube). 

Install Kube Prometheus https://github.com/prometheus-operator/kube-prometheus#quickstart

Await while all pods in the `monitoring` namespace are running. 

You can expose Prometheus dashboard: https://github.com/prometheus-operator/kube-prometheus#access-the-dashboards or use ingresses for this purpose.

Provide more permissions for Prometheuse to watch metrics in the another namespaces:

```bash
kubectl patch clusterrole prometheus-k8s --type='json' -p='[
  {"op": "add", "path": "/rules/0", "value": {"apiGroups": [""], "resources": ["services", "endpoints", "pods"], "verbs": ["get","list","watch"]}},
  {"op": "add", "path": "/rules/0", "value": {"apiGroups": ["extensions"], "resources": ["ingresses"], "verbs": ["get","list","watch"]}},
  {"op": "add", "path": "/rules/0", "value": {"apiGroups": ["networking.k8s.io"], "resources": ["ingresses"], "verbs": ["get","list","watch"]}}
]'
```

Configure Prometheus to find kubernetes-image-puller operator metrics by labels:

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

Apply kubernetes-image-puller ServiceMonitor:

```bash
kubectl apply -f config/prometheus/monitor.yaml
```

Open Prometheus dashboard and type in the "Expression" input:

```
{namespace="kubernetes-image-puller-operator"}
```

Click `Execute` button. Metrics logs should appear in the "Table" tab. Also you can check monitoring targets: click Status and Targets in the dropdown.

### Development

#### Prequisites
* Go >=`1.15`
* Operator SDK `v1.9.2` (recommended)

#### Check code compilation

Run VSCode task `Compile code` or use the terminal:

```bash
$ make compile
```

#### Unit testing

Run VSCode task `Run unit tests` or use the terminal:

```bash
$ make test
```

#### Format code

Run VSCode task `Format code` or use the terminal:

```bash
$ make fmt
```

#### Update golang dependencies

This project uses Go modules and doesn't use vendor folder. Run the VSCode task: `Update dependencies` or use the terminal:

```bash
$ go mod tidy
```

#### Building custom operator image

Use the terminal:

```bash
$ make docker-build docker-push IMG=<CUSTOM_IMAGE>
```

#### Installing using make

Use the terminal:

```bash
$ make deploy IMG=<CUSTOM_IMAGE>
$ kubectl apply -f config/samples/che_v1alpha1_kubernetesimagepuller.yaml -n <NAMESPACE>
```

To uninstall operator:

```bash
$ make undeploy
```

#### Update CR/CRD

Run VSCode task `Update CR/CRDs` or use the terminal:

```bash
$ make generate manifests
```

#### Update OLM bundle

Run VSCode task `Update OLM bundle` or use the terminal:

```bash
$ make bundle
```

#### Releasing a new version of the operator to OperatorHub

A quirk of this project is that while the repository is named `kubernetes-image-puller-operator`, 
the operator bundle on OperatorHub is named `kubernetes-imagepuller-operator`.  
his was caused by the previous version of OLM deployment that required a Quay.io Application.  

Make release bundle:

```bash
$ olm-catalog/release-catalog.sh --version <RELEASE-VERSION>
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

### Trademark

"Che" is a trademark of the Eclipse Foundation.

