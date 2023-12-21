# CHANGELOG

## v1.1.0
- [Introduce webhook to prevent more than 1 KIP resource in a single namespace #115](https://github.com/che-incubator/kubernetes-image-puller-operator/pull/115)
    - The Kubernetes Image Puller Operator now deploys a validating webhook to prevent more than one `KubernetesImagePuller` custom resource from being created in the same namespace.
    - For Kubernetes clusters, [cert-manager](https://github.com/cert-manager/cert-manager) operator must be installed in order to serve the validating webhook.
    - Names of resources that are created alongside the Kubernetes Image Puller Operator installation has been renamed.
        - The `controller-manager-metrics-service` Service has been renamed to `kubernetes-image-puller-operator-manager-metrics-service`
        - The `metrics-reader` ClusterRole has been renamed to `kubernetes-image-puller-operator-metrics-reader`
        - The `kubernetes-image-puller-operator` ServiceAccount has been renamed to `kubernetes-image-puller-operator-sa`
        - The ` kubernetes-image-puller-operator` Deployment has been renamed to `kubernetes-image-puller-operator-manager`
