![Next Build Status](https://github.com/che-incubator/kubernetes-image-puller-operator/actions/workflows/next-build.yml/badge.svg)
![Release Build Status](https://github.com/che-incubator/kubernetes-image-puller-operator/actions/workflows/release.yml/badge.svg)

# Kubernetes Image Puller Operator

[![Contribute](https://www.eclipse.org/che/contribute.svg)](https://workspaces.openshift.com#https://github.com/che-incubator/kubernetes-image-puller-operator)

An operator to install, configure, and manage a [kubernetes-image-puller](https://github.com/che-incubator/kubernetes-image-puller) deployment.

The kubernetes-image-puller creates daemonsets that will run a list of images on a cluster, 
allowing Eclipse Che to start workspaces faster, because those images have been pre-pulled.  
For more information about the kubernetes-image-puller, consult the kubernetes-image-puller 
[README](https://github.com/che-incubator/kubernetes-image-puller/blob/master/README.md).

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
    che-code=quay.io/che-incubator/che-code:next;base-developer-image=quay.io/devfile/base-developer-image:ubi9-latest
  cachingIntervalHours: '2' # number of hours between health checks
  cachingMemoryRequest: '10Mi' # the memory request for each pre-pulled image
  cachingMemoryLimit: '20Mi' # the memory limit for each pre-pulled image
  nodeSelector: '{}' # node selector applied to pods created by the daemonset
```

### Installing The Operator

> Notice: starting from version 1.1.0, this operator requires the cert-manager operator for serving validating webhooks on Kubernetes clusters. This is not required for installation on OpenShift clusters. To install cert-manager on a Kubernetes cluster run:
```
kubectl apply --validate=false -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
```

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

To uninstall operator use commands:

```bash
$ make undeploy
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
* Operator SDK `v1.9.2`

#### Check code compilation

Run the VSCode task `Compile code` or use the terminal:

```bash
$ make compile
```

#### Unit testing

Run the VSCode task `Run unit tests` or use the terminal:

```bash
$ make test
```

#### Format code

Run the VSCode task `Format code` or use the terminal:

```bash
$ make fmt
```

#### Update golang dependencies

This project uses Go modules and doesn't use a vendor folder. Run the VSCode task: `Update dependencies` or use the terminal:

```bash
$ go mod tidy
```

#### Update Kubernetes resources

```bash
$ make bundle
```

#### Building custom operator image

```bash
$ make docker-build docker-push IMG=<CUSTOM_IMAGE>
```

#### Installing using make

```bash
$ make docker-build docker-push IMG=<CUSTOM_IMAGE>
$ make deploy IMG=<CUSTOM_IMAGE>
$ kubectl apply -f config/samples/che_v1alpha1_kubernetesimagepuller.yaml -n kubernetes-image-puller-operator
```

To uninstall the operator:

```bash
$ make undeploy
```

#### Installing using OLM

```bash
$ make bundle
$ bundle/test-bundle.sh
$ kubectl apply -f config/samples/che_v1alpha1_kubernetesimagepuller.yaml -n kubernetes-image-puller-operator
```

#### Releasing a new version of the operator to OperatorHub

A quirk of this project is that while the repository is named `kubernetes-image-puller-operator`, 
the operator bundle on OperatorHub is named `kubernetes-imagepuller-operator`.  
This was caused by the previous version of OLM deployment that required a Quay.io Application.

##### Step 1: Run the Release GitHub Action

Run the [`Release Kubernetes Image Puller operator`](https://github.com/che-incubator/kubernetes-image-puller-operator/actions/workflows/release.yml) GitHub Action with the following inputs:

| Input | Description | Example                                         |
|---|---|-------------------------------------------------|
| `version` | Release version in `X.Y.Z` format | `1.1.2`                                         |
| `imagePullerImage` | kubernetes-image-puller image to use | `quay.io/eclipse/kubernetes-image-puller:1.1.2` |
| `forceRecreateTags` | Recreate existing tags (use with caution) | `false`                                         |

##### Step 2: Checkout to the release branch

After the GitHub Action completes, checkout to the release branch:

```bash
git fetch origin
git checkout <RELEASE_VERSION>-release
```

##### Step 3: Test on Kubernetes and OpenShift

Before publishing to OperatorHub, verify the release on both platforms. [`hack/e2e.sh`](hack/e2e.sh) runs the full verification and works against any cluster (`make e2e` is a shortcut for it):

- the operator deployment becomes available
- a `KubernetesImagePuller` reaches `Ready`
- the ConfigMap, Deployment and DaemonSet are created and the DaemonSet pods are scheduled and ready
- updating the CR propagates to the ConfigMap and rolls the image puller Deployment
- a second `KubernetesImagePuller` in the same namespace is rejected by the validating webhook
- deleting the CR removes every owned resource

Run the verification on a **clean** cluster: a second installation registers another cluster-scoped webhook, and the script refuses to run when one is already present.

**Automated run:** the [`E2E`](.github/workflows/e2e.yml) workflow runs the verification on minikube for pull requests that touch the operator, and on demand. When dispatching it, pass the release branch or tag as `ref`, and optionally a published `bundle-image` to also verify the OLM installation path. Leave `bundle-image` empty to verify only the kustomize manifests.

**Manual run on Kubernetes:**

```bash
kubectl apply --validate=false -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
kubectl wait --for=condition=Available deployment --all -n cert-manager --timeout=300s
./hack/e2e.sh --install-method deploy
```

**Manual run on OpenShift** (cert-manager is not needed, OpenShift signs the webhook certificates itself):

```bash
oc login --server=<SERVER> --token=<TOKEN>
./hack/e2e.sh --install-method deploy
```

To verify the published OLM bundle instead of the manifests:

```bash
make e2e E2E_INSTALL_METHOD=olm \
  E2E_ARGS="--bundle-image quay.io/eclipse/kubernetes-image-puller-operator-bundle:<RELEASE_VERSION>"
```

##### Step 4: Publish to OperatorHub

The [`Publish to OperatorHub`](.github/workflows/publish-operatorhub.yml) workflow automates this step. From the `<RELEASE_VERSION>-release` branch it ensures forks of both repositories exist, copies the bundle, commits it with DCO sign-off and opens (or updates) the pull requests to [`community-operators`](https://github.com/k8s-operatorhub/community-operators/) and [`community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/).

One-off prerequisites — the workflow fails without them:

- [ ] Create a personal access token. A classic token with the `repo` scope is the simplest option; a fine-grained token needs `Contents: Read and write` and `Pull requests: Read and write` on both OperatorHub repositories. The default `GITHUB_TOKEN` cannot push to your forks or open pull requests in other repositories.
- [ ] Store it as the `OPERATORHUB_TOKEN` repository secret.
- [ ] Set the `OPERATORHUB_GIT_NAME` and `OPERATORHUB_GIT_EMAIL` repository variables. They are used as the commit author and for the DCO sign-off, and must match the account that owns the token.
- [ ] Make sure that account can push to its forks of both repositories. The workflow creates missing forks itself.
- [ ] Dispatch the workflow once with `dry-run` enabled and review the commits it reports, then run it again with `dry-run` disabled. The `ref` input defaults to `<RELEASE_VERSION>-release`, so it must contain the bundle for that version.

**Manual equivalent**, if you prefer to run it locally:

```bash
./hack/publish-operatorhub.sh --version <RELEASE_VERSION> --dry-run   # prepare the commits and review them
./hack/publish-operatorhub.sh --version <RELEASE_VERSION>             # push to your forks and open the pull requests
```

The script wraps the older, fully manual flow, which it replaces:

1. Fork and clone the [`community-operators`](https://github.com/k8s-operatorhub/community-operators) 
and [`community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/) repositories.
2. Copy the new bundle into both repositories:
```bash
./make-release.sh <RELEASE_VERSION> --prepare-community-operators-update --community-operators-repository-dir <PROJECT_DIR>/community-operators-prod
./make-release.sh <RELEASE_VERSION> --prepare-community-operators-update --community-operators-repository-dir <PROJECT_DIR>/community-operators
```
3. Commit the new bundle with `git commit -s`, push it to your fork and open two separate pull requests. Examples of the PRs:
- https://github.com/k8s-operatorhub/community-operators/pull/8764
- https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/10447

### Trademark

"Che" is a trademark of the Eclipse Foundation.

