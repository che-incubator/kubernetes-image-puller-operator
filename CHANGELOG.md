# CHANGELOG

## v1.1.3
- [Add status conditions (Ready/Progressing/Degraded) to KubernetesImagePuller CR #237](https://github.com/che-incubator/kubernetes-image-puller-operator/pull/237)
    - The operator now reports structured status conditions on the `KubernetesImagePuller` custom resource, making it easier to monitor rollout progress and diagnose failures.
- [Emit Kubernetes Events on status condition transitions #251](https://github.com/che-incubator/kubernetes-image-puller-operator/pull/251)
    - Status condition changes are now surfaced as Kubernetes Events, visible via `kubectl describe` and cluster event streams.
- [Use annotation-triggered rollout instead of deleting pods #202](https://github.com/che-incubator/kubernetes-image-puller-operator/pull/202)
    - ConfigMap changes now trigger a rolling restart via deployment annotation rather than directly deleting pods, giving Kubernetes control over the rollout.
- [Add update paths for RBAC resources on drift #243](https://github.com/che-incubator/kubernetes-image-puller-operator/pull/243)
    - The operator now reconciles Role and RoleBinding resources back to the desired state if they drift, instead of only creating them.
- [Add delay to reconcile requeue to prevent hot-looping #260](https://github.com/che-incubator/kubernetes-image-puller-operator/pull/260)
    - Requeue delay added to prevent the controller from hot-looping when waiting for resources to converge.
- [Remove dead ConfigMap rename code and rely on cleanup path #201](https://github.com/che-incubator/kubernetes-image-puller-operator/pull/201)
- Bump `kube-rbac-proxy` sidecar from `v0.20.0` to `v0.23.1`
- Dependency updates: controller-runtime, k8s client libraries, ginkgo, gomega, and CI actions

## v1.1.0
- [Introduce webhook to prevent more than 1 KIP resource in a single namespace #115](https://github.com/che-incubator/kubernetes-image-puller-operator/pull/115)
    - The Kubernetes Image Puller Operator now deploys a validating webhook to prevent more than one `KubernetesImagePuller` custom resource from being created in the same namespace.
    - For Kubernetes clusters, [cert-manager](https://github.com/cert-manager/cert-manager) operator must be installed in order to serve the validating webhook.
    - Names of resources that are created alongside the Kubernetes Image Puller Operator installation has been renamed.
        - The `controller-manager-metrics-service` Service has been renamed to `kubernetes-image-puller-operator-manager-metrics-service`
        - The `metrics-reader` ClusterRole has been renamed to `kubernetes-image-puller-operator-metrics-reader`
        - The `kubernetes-image-puller-operator` ServiceAccount has been renamed to `kubernetes-image-puller-operator-sa`
        - The ` kubernetes-image-puller-operator` Deployment has been renamed to `kubernetes-image-puller-operator-manager`
