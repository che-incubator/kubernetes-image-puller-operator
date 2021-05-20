package config

import (
	"reflect"

	chev1alpha1 "github.com/che-incubator/kubernetes-image-puller-operator/pkg/apis/che/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewImagePullerConfigMap(instance *chev1alpha1.KubernetesImagePuller) *corev1.ConfigMap {
	defaultConfigMap := DefaultImagePullerConfigMap(instance.Namespace, instance.Spec.ConfigMapName)
	newConfigMap := mergeConfigMapWithCR(instance, defaultConfigMap)
	newConfigMap.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(instance, chev1alpha1.SchemeGroupVersion.WithKind("KubernetesImagePuller")),
	}
	return newConfigMap
}

func DefaultImagePullerConfigMap(namespace string, name string) *corev1.ConfigMap {
	configMapName := "k8s-image-puller"
	if name != "" {
		configMapName = name
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "kubernetes-image-puller",
			},
		},
		Data: map[string]string{
			"DAEMONSET_NAME":         "kubernetes-image-puller",
			"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:nightly;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
			"CACHING_INTERVAL_HOURS": "1",
			"CACHING_MEMORY_REQUEST": "10Mi",
			"CACHING_MEMORY_LIMIT":   "20Mi",
			"CACHING_CPU_REQUEST":    ".05",
			"CACHING_CPU_LIMIT":      ".2",
			"NODE_SELECTOR":          "{}",
			"IMAGE_PULL_SECRETS":     "",
			"AFFINITY":               "{}",
			"NAMESPACE":              namespace,
		},
	}
}

func ConfigMapsDiffer(new, old *corev1.ConfigMap) bool {
	if reflect.DeepEqual(new.Data, old.Data) && new.Name == old.Name {
		return false
	}
	return true
}

func mergeConfigMapWithCR(instance *chev1alpha1.KubernetesImagePuller, defaultConfigMap *corev1.ConfigMap) *corev1.ConfigMap {
	if instance.Spec.Images != "" {
		defaultConfigMap.Data["IMAGES"] = instance.Spec.Images
	}
	if instance.Spec.CachingIntervalHours != "" {
		defaultConfigMap.Data["CACHING_INTERVAL_HOURS"] = instance.Spec.CachingIntervalHours
	}
	if instance.Spec.CachingMemoryRequest != "" {
		defaultConfigMap.Data["CACHING_MEMORY_REQUEST"] = instance.Spec.CachingMemoryRequest
	}
	if instance.Spec.CachingMemoryLimit != "" {
		defaultConfigMap.Data["CACHING_MEMORY_LIMIT"] = instance.Spec.CachingMemoryLimit
	}
	if instance.Spec.CachingCpuRequest != "" {
		defaultConfigMap.Data["CACHING_CPU_REQUEST"] = instance.Spec.CachingCpuRequest
	}
	if instance.Spec.CachingCpuLimit != "" {
		defaultConfigMap.Data["CACHING_CPU_LIMIT"] = instance.Spec.CachingCpuLimit
	}
	if instance.Spec.NodeSelector != "" {
		defaultConfigMap.Data["NODE_SELECTOR"] = instance.Spec.NodeSelector
	}
	if instance.Spec.ImagePullSecrets != "" {
		defaultConfigMap.Data["IMAGE_PULL_SECRETS"] = instance.Spec.ImagePullSecrets
	}
	if instance.Spec.Affinity != "" {
		defaultConfigMap.Data["AFFINITY"] = instance.Spec.Affinity
	}
	if instance.Spec.DaemonsetName != "" {
		defaultConfigMap.Data["DAEMONSET_NAME"] = instance.Spec.DaemonsetName
	}
	return defaultConfigMap
}
