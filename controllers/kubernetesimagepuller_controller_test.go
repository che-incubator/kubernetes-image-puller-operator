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
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		APIVersion:         chev1alpha1.SchemeBuilder.GroupVersion.String(),
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
	defaultConfigMapName    = "k8s-image-puller"
	defaultDeploymentName   = "kubernetes-image-puller"
	defaultImagePullerImage = "quay.io/eclipse/kubernetes-image-puller:next"
)

func defaultImagePuller() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "0",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{},
	}
}

func defaultImagePullerWithConfigMapName() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "1",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{
			ConfigMapName: defaultConfigMapName,
		},
	}
}

func defaultImagePullerWithConfigMapAndDeploymentName() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
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

func defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage() *chev1alpha1.KubernetesImagePuller {
	return &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "2",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{
			ConfigMapName:    defaultConfigMapName,
			DeploymentName:   defaultDeploymentName,
			ImagePullerImage: defaultImagePullerImage,
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

func TestSetsDeploymentName(t *testing.T) {
	client := setupClient(t, defaultImagePullerWithConfigMapName())
	got := &chev1alpha1.KubernetesImagePuller{}
	want := defaultImagePullerWithConfigMapAndDeploymentName()

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

func TestSetsImagePullerImage(t *testing.T) {
	client := setupClient(t, defaultImagePullerWithConfigMapAndDeploymentName())
	got := &chev1alpha1.KubernetesImagePuller{}
	want := &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-puller",
			Namespace:       namespace,
			ResourceVersion: "3",
		},
		Spec: chev1alpha1.KubernetesImagePullerSpec{
			ConfigMapName:    defaultConfigMapName,
			DeploymentName:   defaultDeploymentName,
			ImagePullerImage: defaultImagePullerImage,
		},
	}

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

func TestCreatesRole(t *testing.T) {
	type testcase struct {
		name string
		cr   *chev1alpha1.KubernetesImagePuller
		want *rbacv1.Role
		got  *rbacv1.Role
	}

	for _, tc := range []testcase{{
		name: "default",
		cr:   defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage(),
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
		cr:   defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage(),
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
		cr:   defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage(),
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

func TestCreatesDeployment(t *testing.T) {
	client := setupClient(t, defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage(),
		expectedConfigMap(defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage()),
		createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
	got := &appsv1.Deployment{}
	want := expectedDeployment(defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage())
	r := &KubernetesImagePullerReconciler{
		Client: client,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controllers").WithName("kubernetesimagepuller"),
	}

	_, err := r.Reconcile(context.TODO(), ctrl.Request{NamespacedName: key})
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

func TestCreatesDeploymentWithDifferentImage(t *testing.T) {
	cr := &chev1alpha1.KubernetesImagePuller{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesImagePuller",
			APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
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
				APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-puller",
				Namespace:       namespace,
				ResourceVersion: "0",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:    defaultConfigMapName,
				DeploymentName:   defaultDeploymentName,
				ImagePullerImage: defaultImagePullerImage,
			},
		},
		want: &chev1alpha1.KubernetesImagePuller{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KubernetesImagePuller",
				APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-puller",
				Namespace:       namespace,
				ResourceVersion: "1",
			},
			Spec: chev1alpha1.KubernetesImagePullerSpec{
				ConfigMapName:    defaultConfigMapName,
				DeploymentName:   defaultDeploymentName,
				ImagePullerImage: defaultImagePullerImage,
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
				APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
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
			TypeMeta: metav1.TypeMeta{
				Kind:       "KubernetesImagePuller",
				APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
			},
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
		cr:   defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage(),
		want: expectedConfigMap(defaultImagePuller()),
		got:  &corev1.ConfigMap{},
	},
		{
			name: "different daemonset name",
			cr: &chev1alpha1.KubernetesImagePuller{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubernetesImagePuller",
					APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					DaemonsetName:    "other-daemonset-name",
					ConfigMapName:    defaultConfigMapName,
					DeploymentName:   defaultDeploymentName,
					ImagePullerImage: defaultImagePullerImage,
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
					"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:next;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            defaultConfigMapName,
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
					APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					ConfigMapName:    defaultConfigMapName,
					DeploymentName:   defaultDeploymentName,
					DaemonsetName:    "other-daemonset-name",
					ImagePullerImage: defaultImagePullerImage,
					Images:           "che-devfile-registry=quay.io/eclipse/che-devfile-registry:latest,woopra-backend=quay.io/openshiftio/che-workspace-telemetry-woopra-backend:latest",
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
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            defaultConfigMapName,
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
					APIVersion: chev1alpha1.SchemeBuilder.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-puller",
					Namespace: namespace,
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					ConfigMapName:    "my-configmap",
					DeploymentName:   defaultDeploymentName,
					ImagePullerImage: defaultImagePullerImage,
				},
			},
			want: &corev1.ConfigMap{
				Data: map[string]string{
					"CACHING_INTERVAL_HOURS": "1",
					"CACHING_MEMORY_LIMIT":   "20Mi",
					"CACHING_MEMORY_REQUEST": "10Mi",
					"CACHING_CPU_LIMIT":      ".2",
					"CACHING_CPU_REQUEST":    ".05",
					"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:next;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
					"DAEMONSET_NAME":         "kubernetes-image-puller",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"KIP_IMAGE":              defaultImagePullerImage,
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
					DaemonsetName:    "new-daemonset",
					ConfigMapName:    defaultConfigMapName,
					DeploymentName:   defaultDeploymentName,
					ImagePullerImage: defaultImagePullerImage,
				},
			},
			old: expectedConfigMap(defaultImagePullerWithConfigMapNameDeploymentNameAndImagePullerImage()),
			want: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            defaultConfigMapName,
					ResourceVersion: "2",
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
					"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:next;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
			},
			got: &corev1.ConfigMap{},
		},
		{
			name: "change the configmap name",
			old:  expectedConfigMap(defaultImagePullerWithConfigMapAndDeploymentName()),
			cr: &chev1alpha1.KubernetesImagePuller{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-puller",
					// ResourceVersion: "0",
				},
				Spec: chev1alpha1.KubernetesImagePullerSpec{
					ConfigMapName:    "new-configmap",
					DeploymentName:   defaultDeploymentName,
					ImagePullerImage: defaultImagePullerImage,
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
					"IMAGES":                 "java11-maven=quay.io/eclipse/che-java11-maven:next;che-theia=quay.io/eclipse/che-theia:next;java-plugin-runner=eclipse/che-remote-plugin-runner-java8:latest",
					"NODE_SELECTOR":          "{}",
					"IMAGE_PULL_SECRETS":     "",
					"AFFINITY":               "{}",
					"KIP_IMAGE":              defaultImagePullerImage,
					"NAMESPACE":              "test",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := setupClient(t, tc.cr, tc.old, NewImagePullerDeployment(tc.cr), createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount)
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
				ConfigMapName:  defaultConfigMapName,
				DeploymentName: "new-kubernetes-image-puller",
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			c := setupClient(t, tc.newCr, NewImagePullerDeployment(tc.newCr), createDaemonsetRole, createDaemonsetRoleBinding, defaultServiceAccount, expectedConfigMap(tc.newCr))
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
