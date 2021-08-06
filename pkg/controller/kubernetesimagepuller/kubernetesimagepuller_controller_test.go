package kubernetesimagepuller

import (
	"context"
	"testing"

	chev1alpha1 "github.com/che-incubator/kubernetes-image-puller-operator/pkg/apis/che/v1alpha1"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/config"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	namespace = "test"
	key       = types.NamespacedName{
		Namespace: namespace,
		Name:      "test-puller",
	}
	isController            = true
	isBlockOwnerDeletion    = true
	defaultCROwnerReference = metav1.OwnerReference{
		APIVersion:         chev1alpha1.SchemeGroupVersion.String(),
		Kind:               "KubernetesImagePuller",
		Name:               "test-puller",
		Controller:         &isController,
		BlockOwnerDeletion: &isBlockOwnerDeletion,
	}
	createDaemonsetRole = &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "create-daemonset",
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
			ResourceVersion: "1",
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"apps"},
			Resources: []string{"daemonsets", "deployments"},
			Verbs:     []string{"create", "delete", "list", "watch", "get"},
		}},
	}
	createDaemonsetRoleBinding = &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "create-daemonset",
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
			ResourceVersion: "1",
		},
		Subjects: []rbacv1.Subject{{
			Kind: "ServiceAccount",
			Name: "k8s-image-puller",
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     "create-daemonset",
		},
	}
	defaultServiceAccount = &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "k8s-image-puller",
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
			ResourceVersion: "1",
		},
	}
)

func defaultImagePuller() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-puller",
			Namespace: namespace,
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{},
	}
}

func defaultImagePullerWithConfigMapName() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "1",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{
			ConfigMapName: "k8s-image-puller",
		},
	}
}

func defaultImagePullerWithConfigMapAndDeploymentName() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "2",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{
			ConfigMapName:  "k8s-image-puller",
			DeploymentName: "kubernetes-image-puller",
		},
	}
}

func expectedDeployment(cr *chev1alpha1.KubernetesImagePuller) *appsv1.Deployment {
	deployment := NewImagePullerDeployment(cr)
	deployment.ResourceVersion = "1"
	deployment.TypeMeta = metav1.TypeMeta{
		Kind:       "Deployment",
		APIVersion: appsv1.SchemeGroupVersion.String(),
	}
	return deployment
}

func expectedConfigMap(cr *chev1alpha1.KubernetesImagePuller) *corev1.ConfigMap {
	configMap := config.NewImagePullerConfigMap(cr)
	configMap.ObjectMeta.OwnerReferences = []metav1.OwnerReference{defaultCROwnerReference}
	configMap.ResourceVersion = "1"
	return configMap
}

func setupClient(t *testing.T, objs ...runtime.Object) client.Client {
	if err := chev1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Errorf("Error adding to scheme: %v", err)
	}
	client := fake.NewFakeClientWithScheme(scheme.Scheme, objs...)
	return client
}

func TestSetsConfigMapName(t *testing.T) {
	client := setupClient(t, defaultImagePuller())
	got := &chev1alpha1.KubernetesImagePuller{}
	want := defaultImagePullerWithConfigMapName()

	r := &ReconcileKubernetesImagePuller{
		client: client,
		scheme: scheme.Scheme,
	}

	_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
	if err != nil {
		t.Errorf("Got error in reconcile: %v", err)
	}

	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "test-puller"}, got); err != nil {
		t.Errorf("Error getting KubernetesImagePuller")
	}

	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("Error (-want, +got): %s", d)
	}
}

func TestSetsDeploymentName(t *testing.T) {
	client := setupClient(t, defaultImagePullerWithConfigMapName())
	got := &chev1alpha1.KubernetesImagePuller{}
	want := defaultImagePullerWithConfigMapAndDeploymentName()

	r := &ReconcileKubernetesImagePuller{
		client: client,
		scheme: scheme.Scheme,
	}

	_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
	if err != nil {
		t.Errorf("Got error in reconcile: %v", err)
	}

	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "test-puller"}, got); err != nil {
		t.Errorf("Error getting KubernetesImagePuller")
	}

	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("Error (-want, +got): %s", d)
	}
}

func TestCreatesRole(t *testing.T) {
	type testcase struct {
		name string
		cr   *chev1alpha1.KubernetesImagePuller
		want *rbacv1.Role
		got  *rbacv1.Role
	}

	for _, tc := range []testcase{{
		name: "default",
		cr:   defaultImagePullerWithConfigMapAndDeploymentName(),
		want: createDaemonsetRole,
		got:  &rbacv1.Role{},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr)
			r := &ReconcileKubernetesImagePuller{
				client: client,
				scheme: scheme.Scheme,
			}

			_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Error in reconcile: %v", err)
			}

			if err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: createDaemonsetRole.Name}, tc.got); err != nil {
				t.Errorf("Error getting Role: %v", err)
			}

			if d := cmp.Diff(tc.want, tc.got); d != "" {
				t.Errorf("Error (-want, +got): %s", d)
			}
		})
	}
}

func TestCreatesRoleBinding(t *testing.T) {
	type testcase struct {
		name string
		cr   *chev1alpha1.KubernetesImagePuller
		want *rbacv1.RoleBinding
		got  *rbacv1.RoleBinding
	}

	for _, tc := range []testcase{{
		name: "default",
		cr:   defaultImagePullerWithConfigMapAndDeploymentName(),
		want: createDaemonsetRoleBinding,
		got:  &rbacv1.RoleBinding{},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, createDaemonsetRole)
			r := &ReconcileKubernetesImagePuller{
				client: client,
				scheme: scheme.Scheme,
			}

			_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Error in reconcile: %v", err)
			}

			if err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: createDaemonsetRoleBinding.Name}, tc.got); err != nil {
				t.Errorf("Error getting RoleBinding: %v", err)
			}

			if d := cmp.Diff(tc.want, tc.got); d != "" {
				t.Errorf("Error (-want, +got): %s", d)
			}
		})
	}
}

func TestCreatesServiceAccount(t *testing.T) {
	type testcase struct {
		name string
		cr   *chev1alpha1.KubernetesImagePuller
		want *corev1.ServiceAccount
		got  *corev1.ServiceAccount
	}

	for _, tc := range []testcase{{
		name: "default",
		cr:   defaultImagePullerWithConfigMapAndDeploymentName(),
		want: defaultServiceAccount,
		got:  &corev1.ServiceAccount{},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, createDaemonsetRole, createDaemonsetRoleBinding)
			r := &ReconcileKubernetesImagePuller{
				client: client,
				scheme: scheme.Scheme,
			}

			_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Error in reconcile: %v", err)
			}

			if err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaultServiceAccount.Name}, tc.got); err != nil {
				t.Errorf("Error getting ServiceAccount: %v", err)
			}

			if d := cmp.Diff(tc.want, tc.got); d != "" {
				t.Errorf("Error (-want, +got): %s", d)
			}
		})
	}
}

func TestCreatesDeployment(t *testing.T) {
	client := setupClient(t, defaultImagePullerWithConfigMapAndDeploymentName(), expectedConfigMap(defaultImagePullerWithConfigMapAndDeploymentName()), createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
	got := &appsv1.Deployment{}
	want := expectedDeployment(defaultImagePullerWithConfigMapAndDeploymentName())
	r := &ReconcileKubernetesImagePuller{
		client: client,
		scheme: scheme.Scheme,
	}

	_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
	if err != nil {
		t.Errorf("Got error in reconcile: %v", err)
	}

	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "kubernetes-image-puller"}, got); err != nil {
		t.Errorf("Error getting deployment: %v", err)
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("Error (-want, +got): %s", d)
	}
}

func TestCreatesConfigMap(t *testing.T) {

	type testcase struct {
		name string
		cr   *chev1alpha1.KubernetesImagePuller
		want *corev1.ConfigMap
		got  *corev1.ConfigMap
	}

	for _, tc := range []testcase{{
		name: "default",
		cr:   defaultImagePullerWithConfigMapAndDeploymentName(),
		want: expectedConfigMap(defaultImagePuller()),
		got:  &corev1.ConfigMap{},
	},
		{
			name: "different daemonset name",
			cr: &chev1alpha1.KubernetesImagePuller{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubernetesImagePuller",
					APIVersion: chev1alpha1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					DaemonsetName:  "other-daemonset-name",
					ConfigMapName:  "k8s-image-puller",
					DeploymentName: "kubernetes-image-puller",
				},
			},
			want: &corev1.ConfigMap{
				Data: map[string]string{
					"CACHING_INTERVAL_HOURS": "1",
					"CACHING_MEMORY_LIMIT":   "20Mi",
					"CACHING_MEMORY_REQUEST": "10Mi",
					"CACHING_CPU_LIMIT":      ".2",
					"CACHING_CPU_REQUEST":    ".05",
					"DAEMONSET_NAME":         "other-daemonset-name",
					"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:nightly;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"KIP_IMAGE":              "quay.io/eclipse/kubernetes-image-puller:next",
					"NAMESPACE":              "test",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            "k8s-image-puller",
					ResourceVersion: "1",
					Labels:          map[string]string{"app": "kubernetes-image-puller"},
					OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
			},
			got: &corev1.ConfigMap{},
		},
		{
			name: "different daemonset and images",
			cr: &chev1alpha1.KubernetesImagePuller{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubernetesImagePuller",
					APIVersion: chev1alpha1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					ConfigMapName:  "k8s-image-puller",
					DeploymentName: "kubernetes-image-puller",
					DaemonsetName:  "other-daemonset-name",
					Images:         "che-devfile-registry=quay.io/eclipse/che-devfile-registry:latest,woopra-backend=quay.io/openshiftio/che-workspace-telemetry-woopra-backend:latest",
				},
			},
			want: &corev1.ConfigMap{
				Data: map[string]string{
					"CACHING_INTERVAL_HOURS": "1",
					"CACHING_MEMORY_LIMIT":   "20Mi",
					"CACHING_MEMORY_REQUEST": "10Mi",
					"CACHING_CPU_LIMIT":      ".2",
					"CACHING_CPU_REQUEST":    ".05",
					"IMAGES":                 "che-devfile-registry=quay.io/eclipse/che-devfile-registry:latest,woopra-backend=quay.io/openshiftio/che-workspace-telemetry-woopra-backend:latest",
					"DAEMONSET_NAME":         "other-daemonset-name",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"KIP_IMAGE":              "quay.io/eclipse/kubernetes-image-puller:next",
					"NAMESPACE":              "test",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            "k8s-image-puller",
					ResourceVersion: "1",
					Labels:          map[string]string{"app": "kubernetes-image-puller"},
					OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
			},
			got: &corev1.ConfigMap{},
		},
		{
			name: "different configmap name",
			cr: &chev1alpha1.KubernetesImagePuller{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubernetesImagePuller",
					APIVersion: chev1alpha1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					ConfigMapName:  "my-configmap",
					DeploymentName: "kubernetes-image-puller",
				},
			},
			want: &corev1.ConfigMap{
				Data: map[string]string{
					"CACHING_INTERVAL_HOURS": "1",
					"CACHING_MEMORY_LIMIT":   "20Mi",
					"CACHING_MEMORY_REQUEST": "10Mi",
					"CACHING_CPU_LIMIT":      ".2",
					"CACHING_CPU_REQUEST":    ".05",
					"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:nightly;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
					"DAEMONSET_NAME":         "kubernetes-image-puller",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"KIP_IMAGE":              "quay.io/eclipse/kubernetes-image-puller:next",
					"NAMESPACE":              "test",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            "my-configmap",
					ResourceVersion: "1",
					Labels:          map[string]string{"app": "kubernetes-image-puller"},
					OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
			},
			got: &corev1.ConfigMap{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
			r := &ReconcileKubernetesImagePuller{
				client: client,
				scheme: scheme.Scheme,
			}
			_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Got error in reconcile: %v", err)
			}

			if tc.cr.Spec.ConfigMapName != "" {
				if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: tc.cr.Spec.ConfigMapName}, tc.got); err != nil {
					t.Errorf("Error getting configmap: %v", err)
				}
			} else {
				if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "k8s-image-puller"}, tc.got); err != nil {
					t.Errorf("Error getting configmap: %v", err)
				}
			}
			if d := cmp.Diff(tc.want, tc.got); d != "" {
				t.Errorf("Error (-want, +got): %s", d)
			}
		})
	}
}

func TestUpdatesConfigMap(t *testing.T) {
	type testcase struct {
		name string
		cr   *chev1alpha1.KubernetesImagePuller
		old  *corev1.ConfigMap
		want *corev1.ConfigMap
		got  *corev1.ConfigMap
	}

	for _, tc := range []testcase{{
		name: "default update",
		cr: &chev1alpha1.KubernetesImagePuller{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "test-puller",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				DaemonsetName:  "new-daemonset",
				ConfigMapName:  "k8s-image-puller",
				DeploymentName: "kubernetes-image-puller",
			},
		},
		old: expectedConfigMap(defaultImagePullerWithConfigMapAndDeploymentName()),
		want: &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       namespace,
				Name:            "k8s-image-puller",
				ResourceVersion: "1",
				Labels:          map[string]string{"app": "kubernetes-image-puller"},
				OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
			},
			Data: map[string]string{
				"CACHING_INTERVAL_HOURS": "1",
				"CACHING_MEMORY_LIMIT":   "20Mi",
				"CACHING_MEMORY_REQUEST": "10Mi",
				"CACHING_CPU_LIMIT":      ".2",
				"CACHING_CPU_REQUEST":    ".05",
				"DAEMONSET_NAME":         "new-daemonset",
				"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:nightly;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
				"NODE_SELECTOR":          "{}",
				"IMAGE_PULL_SECRETS":     "",
				"AFFINITY":               "{}",
				"KIP_IMAGE":              "quay.io/eclipse/kubernetes-image-puller:next",
				"NAMESPACE":              "test",
			},
		},
		got: &corev1.ConfigMap{},
	}, {
		name: "change the configmap name",
		old:  expectedConfigMap(defaultImagePullerWithConfigMapAndDeploymentName()),
		cr: &chev1alpha1.KubernetesImagePuller{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "test-puller",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:  "new-configmap",
				DeploymentName: "kubernetes-image-puller",
			},
		},
		want: &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       namespace,
				Name:            "new-configmap",
				ResourceVersion: "1",
				Labels:          map[string]string{"app": "kubernetes-image-puller"},
				OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
			},
			Data: map[string]string{
				"CACHING_INTERVAL_HOURS": "1",
				"CACHING_MEMORY_LIMIT":   "20Mi",
				"CACHING_MEMORY_REQUEST": "10Mi",
				"CACHING_CPU_LIMIT":      ".2",
				"CACHING_CPU_REQUEST":    ".05",
				"DAEMONSET_NAME":         "kubernetes-image-puller",
				"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:nightly;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
				"NODE_SELECTOR":          "{}",
				"IMAGE_PULL_SECRETS":     "",
				"AFFINITY":               "{}",
				"KIP_IMAGE":              "quay.io/eclipse/kubernetes-image-puller:next",
				"NAMESPACE":              "test",
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, tc.old, NewImagePullerDeployment(tc.cr), createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
			r := &ReconcileKubernetesImagePuller{
				client: client,
				scheme: scheme.Scheme,
			}
			_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Got error in reconcile: %v", err)
			}

			tc.got = &corev1.ConfigMap{}
			if tc.cr.Spec.ConfigMapName != "" {
				if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: tc.cr.Spec.ConfigMapName}, tc.got); err != nil {
					t.Errorf("Error getting configmap: %v", err)
				}
			} else {
				if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "k8s-image-puller"}, tc.got); err != nil {
					t.Errorf("Error getting configmap: %v", err)
				}
			}
			if d := cmp.Diff(tc.want, tc.got); d != "" {
				t.Errorf("Error (-want, +got): %s", d)
			}
		})
	}
}

func TestDeletesOldDeploymentOnNameChange(t *testing.T) {
	type testcase struct {
		name  string
		oldCr *chev1alpha1.KubernetesImagePuller
		newCr *chev1alpha1.KubernetesImagePuller
	}

	for _, tc := range []testcase{{
		name:  "change the deployment name",
		oldCr: defaultImagePullerWithConfigMapAndDeploymentName(),
		newCr: &chev1alpha1.KubernetesImagePuller{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "test-puller",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:  "k8s-image-puller",
				DeploymentName: "new-kubernetes-image-puller",
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			c := setupClient(t, tc.newCr, NewImagePullerDeployment(tc.newCr), createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount, expectedConfigMap(tc.newCr))
			r := &ReconcileKubernetesImagePuller{
				client: c,
				scheme: scheme.Scheme,
			}
			_, err := r.Reconcile(reconcile.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Got error in reconcile: %v", err)
			}
			deployments := &appsv1.DeploymentList{}
			c.List(context.TODO(), deployments, client.MatchingLabels{"app": "kubernetes-image-puller"})
			if err != nil {
				t.Errorf("Error listing deployments: %v", err)
			}
			if len(deployments.Items) != 1 {
				t.Errorf("Expected 1 deployment but got %v", len(deployments.Items))
			}

			if deployments.Items[0].Name != tc.newCr.Spec.DeploymentName {
				t.Errorf("Expected a deployment named %v but got %v", tc.newCr.Spec.DeploymentName, deployments.Items[0].Name)
			}
		})
	}
}
