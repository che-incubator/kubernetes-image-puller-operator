//
// Copyright (c) 2012-2021 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package controllers

import (
	"context"
	"testing"

	chev1alpha1 "github.com/che-incubator/kubernetes-image-puller-operator/api/v1alpha1"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/config"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/defaults"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
		APIVersion:         chev1alpha1.GroupVersion.String(),
		Kind:               "KubernetesImagePuller",
		Name:               "test-puller",
		Controller:         &isController,
		BlockOwnerDeletion: &isBlockOwnerDeletion,
	}
	createDaemonsetRole = &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            defaults.RBACName,
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
		ObjectMeta: metav1.ObjectMeta{
			Name:            defaults.RBACName,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
			ResourceVersion: "1",
		},
		Subjects: []rbacv1.Subject{{
			Kind: "ServiceAccount",
			Name: defaults.ServiceAccountName,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     defaults.RBACName,
		},
	}
	defaultServiceAccount = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:            defaults.ServiceAccountName,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
			ResourceVersion: "1",
		},
	}
	defaultConfigMapName    = defaults.ConfigMapName
	defaultDeploymentName   = defaults.DeploymentName
	defaultImagePullerImage = defaults.ImagePullerImage
)

func defaultImagePuller() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "0",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{},
	}
}

func defaultImagePullerWithAllDefaults() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "1",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{
			ConfigMapName:  defaultConfigMapName,
			DeploymentName: defaultDeploymentName,
		},
	}
}

func defaultImagePullerWithConfigMapNameAndDeploymentName() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "2",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{
			ConfigMapName:  defaultConfigMapName,
			DeploymentName: defaultDeploymentName,
		},
	}
}

func expectedDeployment(cr *chev1alpha1.KubernetesImagePuller) *appsv1.Deployment {
	deployment := NewImagePullerDeployment(cr, false)
	deployment.ResourceVersion = "1"
	return deployment
}

func expectedConfigMap(cr *chev1alpha1.KubernetesImagePuller) *corev1.ConfigMap {
	configMap := config.NewImagePullerConfigMap(cr)
	configMap.TypeMeta = metav1.TypeMeta{}
	configMap.OwnerReferences = []metav1.OwnerReference{defaultCROwnerReference}
	configMap.ResourceVersion = "1"
	return configMap
}

func setupClient(t *testing.T, objs ...client.Object) client.Client {
	if err := chev1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Errorf("Error adding to scheme: %v", err)
	}
	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(objs...).
		WithStatusSubresource(&chev1alpha1.KubernetesImagePuller{}).
		Build()
}

func TestSetsAllDefaults(t *testing.T) {
	client := setupClient(t, defaultImagePuller())
	got := &chev1alpha1.KubernetesImagePuller{}
	want := defaultImagePullerWithAllDefaults()

	r := &KubernetesImagePullerReconciler{
		Client: client,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
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

func TestDeploymentUsesDefaultImageWhenSpecEmpty(t *testing.T) {
	cr := defaultImagePullerWithConfigMapNameAndDeploymentName()
	client := setupClient(t, cr,
		expectedConfigMap(cr),
		createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
	r := &KubernetesImagePullerReconciler{
		Client: client,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
	if err != nil {
		t.Errorf("Got error in reconcile: %v", err)
	}

	got := &appsv1.Deployment{}
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaultDeploymentName}, got); err != nil {
		t.Errorf("Error getting deployment: %v", err)
	}

	if got.Spec.Template.Spec.Containers[0].Image != defaultImagePullerImage {
		t.Errorf("Expected deployment image %q, got %q", defaultImagePullerImage, got.Spec.Template.Spec.Containers[0].Image)
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
		cr:   defaultImagePullerWithConfigMapNameAndDeploymentName(),
		want: createDaemonsetRole,
		got:  &rbacv1.Role{},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr)
			r := &KubernetesImagePullerReconciler{
				Client: client,
				Scheme: scheme.Scheme,
				Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
			}

			_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
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
		cr:   defaultImagePullerWithConfigMapNameAndDeploymentName(),
		want: createDaemonsetRoleBinding,
		got:  &rbacv1.RoleBinding{},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, createDaemonsetRole)
			r := &KubernetesImagePullerReconciler{
				Client: client,
				Scheme: scheme.Scheme,
				Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
			}

			_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
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
		cr:   defaultImagePullerWithConfigMapNameAndDeploymentName(),
		want: defaultServiceAccount,
		got:  &corev1.ServiceAccount{},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, createDaemonsetRole, createDaemonsetRoleBinding)
			r := &KubernetesImagePullerReconciler{
				Client: client,
				Scheme: scheme.Scheme,
				Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
			}

			_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
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

func TestUpdatesRoleOnDrift(t *testing.T) {
	cr := defaultImagePullerWithConfigMapNameAndDeploymentName()
	driftedRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            defaults.RBACName,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"apps"},
			Resources: []string{"daemonsets"},
			Verbs:     []string{"create", "delete"},
		}},
	}

	c := setupClient(t, cr, driftedRole, createDaemonsetRoleBinding, defaultServiceAccount,
		expectedConfigMap(cr), expectedDeployment(cr))
	r := &KubernetesImagePullerReconciler{
		Client: c,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	if _, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}

	got := &rbacv1.Role{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaults.RBACName}, got); err != nil {
		t.Fatalf("Error getting Role: %v", err)
	}

	if d := cmp.Diff(createDaemonsetRole.Rules, got.Rules); d != "" {
		t.Errorf("Role rules not corrected (-want, +got): %s", d)
	}
}

func TestUpdatesRoleBindingSubjectsOnDrift(t *testing.T) {
	cr := defaultImagePullerWithConfigMapNameAndDeploymentName()
	driftedRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            defaults.RBACName,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
		},
		Subjects: []rbacv1.Subject{{
			Kind: "ServiceAccount",
			Name: "wrong-service-account",
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     defaults.RBACName,
		},
	}

	c := setupClient(t, cr, createDaemonsetRole, driftedRoleBinding, defaultServiceAccount,
		expectedConfigMap(cr), expectedDeployment(cr))
	r := &KubernetesImagePullerReconciler{
		Client: c,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	if _, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}

	got := &rbacv1.RoleBinding{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaults.RBACName}, got); err != nil {
		t.Fatalf("Error getting RoleBinding: %v", err)
	}

	if d := cmp.Diff(createDaemonsetRoleBinding.Subjects, got.Subjects); d != "" {
		t.Errorf("RoleBinding subjects not corrected (-want, +got): %s", d)
	}
}

func TestDeletesRoleBindingOnRoleRefDrift(t *testing.T) {
	cr := defaultImagePullerWithConfigMapNameAndDeploymentName()
	driftedRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            defaults.RBACName,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
		},
		Subjects: []rbacv1.Subject{{
			Kind: "ServiceAccount",
			Name: defaults.ServiceAccountName,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     "wrong-role",
		},
	}

	c := setupClient(t, cr, createDaemonsetRole, driftedRoleBinding, defaultServiceAccount,
		expectedConfigMap(cr), expectedDeployment(cr))
	r := &KubernetesImagePullerReconciler{
		Client: c,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	result, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
	if err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}
	if result == (ctrl.Result{}) {
		t.Error("Expected requeue after deleting RoleBinding with drifted RoleRef")
	}

	got := &rbacv1.RoleBinding{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaults.RBACName}, got)
	if err == nil {
		t.Error("Expected RoleBinding to be deleted, but it still exists")
	}
}

func TestCreatesDeployment(t *testing.T) {
	client := setupClient(t, defaultImagePullerWithConfigMapNameAndDeploymentName(),
		expectedConfigMap(defaultImagePullerWithConfigMapNameAndDeploymentName()),
		createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
	got := &appsv1.Deployment{}
	want := expectedDeployment(defaultImagePullerWithConfigMapNameAndDeploymentName())
	r := &KubernetesImagePullerReconciler{
		Client: client,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
	if err != nil {
		t.Errorf("Got error in reconcile: %v", err)
	}

	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaults.DeploymentName}, got); err != nil {
		t.Errorf("Error getting deployment: %v", err)
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("Error (-want, +got): %s", d)
	}
}

func TestCreatesDeploymentWithDifferentImage(t *testing.T) {
	cr := &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-puller",
			Namespace: namespace,
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{
			ConfigMapName:    defaultConfigMapName,
			DeploymentName:   defaultDeploymentName,
			ImagePullerImage: "quay.io/eclipse/kubernetes-image-puller:new-image",
		},
	}

	client := setupClient(t, cr, expectedConfigMap(cr),
		createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
	got := &appsv1.Deployment{}
	want := "quay.io/eclipse/kubernetes-image-puller:new-image"
	r := &KubernetesImagePullerReconciler{
		Client: client,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
	if err != nil {
		t.Errorf("Got error in reconcile: %v", err)
	}

	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaultDeploymentName}, got); err != nil {
		t.Errorf("Error getting deployment: %v", err)
	}

	if got.Spec.Template.Spec.Containers[0].Image != want {
		t.Errorf("Error: expected %s, but was %s", want, got.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestUpdatesImagePullerImageStatus(t *testing.T) {

	type testcase struct {
		name string
		cr   *chev1alpha1.KubernetesImagePuller
		want *chev1alpha1.KubernetesImagePuller
	}

	for _, tc := range []testcase{{
		name: "update status for the first time",
		cr: &chev1alpha1.KubernetesImagePuller{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KubernetesImagePuller",
				APIVersion: chev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-puller",
				Namespace:       namespace,
				ResourceVersion: "0",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:  defaultConfigMapName,
				DeploymentName: defaultDeploymentName,
			},
		},
		want: &chev1alpha1.KubernetesImagePuller{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-puller",
				Namespace:       namespace,
				ResourceVersion: "1",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:  defaultConfigMapName,
				DeploymentName: defaultDeploymentName,
			},
			Status: chev1alpha1.KubernetesImagePullerStatus{
				ImagePullerImage: defaultImagePullerImage,
			},
		},
	}, {
		name: "update status",
		cr: &chev1alpha1.KubernetesImagePuller{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KubernetesImagePuller",
				APIVersion: chev1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-puller",
				Namespace:       namespace,
				ResourceVersion: "0",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:    defaultConfigMapName,
				DeploymentName:   defaultDeploymentName,
				ImagePullerImage: "quay.io/eclipse/kubernetes-image-puller:new-image",
			},
			Status: chev1alpha1.KubernetesImagePullerStatus{
				ImagePullerImage: defaultImagePullerImage,
			},
		},
		want: &chev1alpha1.KubernetesImagePuller{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-puller",
				Namespace:       namespace,
				ResourceVersion: "1",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:    defaultConfigMapName,
				DeploymentName:   defaultDeploymentName,
				ImagePullerImage: "quay.io/eclipse/kubernetes-image-puller:new-image",
			},
			Status: chev1alpha1.KubernetesImagePullerStatus{
				ImagePullerImage: "quay.io/eclipse/kubernetes-image-puller:new-image",
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, expectedConfigMap(tc.cr),
				createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount, expectedDeployment(tc.cr))
			r := &KubernetesImagePullerReconciler{
				Client: client,
				Scheme: scheme.Scheme,
				Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
			}

			_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Got error in reconcile: %v", err)
			}

			got := &chev1alpha1.KubernetesImagePuller{}
			if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: "test-puller"}, got); err != nil {
				t.Errorf("Error getting KubernetesImagePuller")
			}

			if d := cmp.Diff(tc.want, got); d != "" {
				t.Errorf("Error (-want, +got): %s", d)
			}
		})
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
		cr:   defaultImagePullerWithConfigMapNameAndDeploymentName(),
		want: expectedConfigMap(defaultImagePuller()),
		got:  &corev1.ConfigMap{},
	},
		{
			name: "different daemonset name",
			cr: &chev1alpha1.KubernetesImagePuller{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubernetesImagePuller",
					APIVersion: chev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					DaemonsetName:  "other-daemonset-name",
					ConfigMapName:  defaultConfigMapName,
					DeploymentName: defaultDeploymentName,
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
					"IMAGES":                 "che-code=quay.io/che-incubator/che-code:next;base-developer-image=quay.io/devfile/base-developer-image:ubi9-latest",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"TOLERATIONS":            "[]",
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            defaultConfigMapName,
					ResourceVersion: "1",
					Labels:          map[string]string{"app": defaults.AppLabelValue},
					OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
				},
			},
			got: &corev1.ConfigMap{},
		},
		{
			name: "different daemonset and images",
			cr: &chev1alpha1.KubernetesImagePuller{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubernetesImagePuller",
					APIVersion: chev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					ConfigMapName:  defaultConfigMapName,
					DeploymentName: defaultDeploymentName,
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
					"TOLERATIONS":            "[]",
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            defaultConfigMapName,
					ResourceVersion: "1",
					Labels:          map[string]string{"app": defaults.AppLabelValue},
					OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
				},
			},
			got: &corev1.ConfigMap{},
		},
		{
			name: "different configmap name",
			cr: &chev1alpha1.KubernetesImagePuller{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubernetesImagePuller",
					APIVersion: chev1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					ConfigMapName:  "my-configmap",
					DeploymentName: defaultDeploymentName,
				},
			},
			want: &corev1.ConfigMap{
				Data: map[string]string{
					"CACHING_INTERVAL_HOURS": "1",
					"CACHING_MEMORY_LIMIT":   "20Mi",
					"CACHING_MEMORY_REQUEST": "10Mi",
					"CACHING_CPU_LIMIT":      ".2",
					"CACHING_CPU_REQUEST":    ".05",
					"IMAGES":                 "che-code=quay.io/che-incubator/che-code:next;base-developer-image=quay.io/devfile/base-developer-image:ubi9-latest",
					"DAEMONSET_NAME":         defaults.DeploymentName,
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"TOLERATIONS":            "[]",
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            "my-configmap",
					ResourceVersion: "1",
					Labels:          map[string]string{"app": defaults.AppLabelValue},
					OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
				},
			},
			got: &corev1.ConfigMap{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
			r := &KubernetesImagePullerReconciler{
				Client: client,
				Scheme: scheme.Scheme,
				Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
			}
			_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Got error in reconcile: %v", err)
			}

			if tc.cr.Spec.ConfigMapName != "" {
				if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: tc.cr.Spec.ConfigMapName}, tc.got); err != nil {
					t.Errorf("Error getting configmap: %v", err)
				}
			} else {
				if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaultConfigMapName}, tc.got); err != nil {
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

	for _, tc := range []testcase{
		{
			name: "default update",
			cr: &chev1alpha1.KubernetesImagePuller{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-puller",
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					DaemonsetName:  "new-daemonset",
					ConfigMapName:  defaultConfigMapName,
					DeploymentName: defaultDeploymentName,
				},
			},
			old: expectedConfigMap(defaultImagePullerWithConfigMapNameAndDeploymentName()),
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            defaultConfigMapName,
					ResourceVersion: "2",
					Labels:          map[string]string{"app": defaults.AppLabelValue},
					OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
				},
				Data: map[string]string{
					"CACHING_INTERVAL_HOURS": "1",
					"CACHING_MEMORY_LIMIT":   "20Mi",
					"CACHING_MEMORY_REQUEST": "10Mi",
					"CACHING_CPU_LIMIT":      ".2",
					"CACHING_CPU_REQUEST":    ".05",
					"DAEMONSET_NAME":         "new-daemonset",
					"IMAGES":                 "che-code=quay.io/che-incubator/che-code:next;base-developer-image=quay.io/devfile/base-developer-image:ubi9-latest",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"TOLERATIONS":            "[]",
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
			},
			got: &corev1.ConfigMap{},
		},
		{
			name: "change the configmap name",
			old:  expectedConfigMap(defaultImagePullerWithAllDefaults()),
			cr: &chev1alpha1.KubernetesImagePuller{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-puller",
					// ResourceVersion: "0",
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					ConfigMapName:  "new-configmap",
					DeploymentName: defaultDeploymentName,
				},
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            "new-configmap",
					ResourceVersion: "1",
					Labels:          map[string]string{"app": defaults.AppLabelValue},
					OwnerReferences: []metav1.OwnerReference{defaultCROwnerReference},
				},
				Data: map[string]string{
					"CACHING_INTERVAL_HOURS": "1",
					"CACHING_MEMORY_LIMIT":   "20Mi",
					"CACHING_MEMORY_REQUEST": "10Mi",
					"CACHING_CPU_LIMIT":      ".2",
					"CACHING_CPU_REQUEST":    ".05",
					"DAEMONSET_NAME":         defaults.DeploymentName,
					"IMAGES":                 "che-code=quay.io/che-incubator/che-code:next;base-developer-image=quay.io/devfile/base-developer-image:ubi9-latest",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"TOLERATIONS":            "[]",
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, tc.old, NewImagePullerDeployment(tc.cr, true), createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
			r := &KubernetesImagePullerReconciler{
				Client: client,
				Scheme: scheme.Scheme,
				Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
			}
			_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Got error in reconcile: %v", err)
			}

			tc.got = &corev1.ConfigMap{}
			if tc.cr.Spec.ConfigMapName != "" {
				if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: tc.cr.Spec.ConfigMapName}, tc.got); err != nil {
					t.Errorf("Error getting configmap: %v", err)
				}
			} else {
				if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaultConfigMapName}, tc.got); err != nil {
					t.Errorf("Error getting configmap: %v", err)
				}
			}
			if d := cmp.Diff(tc.want, tc.got); d != "" {
				t.Errorf("Error (-want, +got): %s", d)
			}
		})
	}
}

func TestAnnotationRolloutOnConfigMapChange(t *testing.T) {
	cr := defaultImagePullerWithConfigMapNameAndDeploymentName()
	cr.Spec.DaemonsetName = "new-daemonset"
	oldConfigMap := expectedConfigMap(defaultImagePullerWithConfigMapNameAndDeploymentName())
	deployment := NewImagePullerDeployment(cr, false)
	deployment.ResourceVersion = "1"

	c := setupClient(t, cr, oldConfigMap, deployment,
		createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
	r := &KubernetesImagePullerReconciler{
		Client: c,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	if _, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}

	got := &appsv1.Deployment{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: defaultDeploymentName}, got); err != nil {
		t.Fatalf("Error getting deployment: %v", err)
	}

	annotations := got.Spec.Template.Annotations
	if annotations == nil {
		t.Fatal("Expected pod template annotations to be set, got nil")
	}
	if _, ok := annotations["che.eclipse.org/configmap-hash"]; !ok {
		t.Error("Expected che.eclipse.org/configmap-hash annotation on pod template")
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
		oldCr: defaultImagePullerWithAllDefaults(),
		newCr: &chev1alpha1.KubernetesImagePuller{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "test-puller",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:  defaultConfigMapName,
				DeploymentName: "new-kubernetes-image-puller",
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			c := setupClient(t, tc.newCr, NewImagePullerDeployment(tc.newCr, true), createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount, expectedConfigMap(tc.newCr))
			r := &KubernetesImagePullerReconciler{
				Client: c,
				Scheme: scheme.Scheme,
				Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
			}
			_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
			if err != nil {
				t.Errorf("Got error in reconcile: %v", err)
			}
			deployments := &appsv1.DeploymentList{}
			if err = c.List(context.TODO(), deployments, client.MatchingLabels{"app": defaults.AppLabelValue}); err != nil {
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
