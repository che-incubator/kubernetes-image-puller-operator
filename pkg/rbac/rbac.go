package rbac

import (
	chev1alpha1 "github.com/che-incubator/kubernetes-image-puller-operator/pkg/apis/che/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewRole(cr *chev1alpha1.KubernetesImagePuller) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "create-daemonset",
			Namespace: cr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, chev1alpha1.SchemeGroupVersion.WithKind("KubernetesImagePuller")),
			},
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"apps"},
			Resources: []string{"daemonsets, deployments"},
			Verbs:     []string{"create", "delete", "watch", "get"},
		}},
	}
}

func NewRoleBinding(cr *chev1alpha1.KubernetesImagePuller) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "create-daemonset",
			Namespace: cr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, chev1alpha1.SchemeGroupVersion.WithKind("KubernetesImagePuller")),
			},
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
}

func NewServiceAccount(cr *chev1alpha1.KubernetesImagePuller) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-image-puller",
			Namespace: cr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, chev1alpha1.SchemeGroupVersion.WithKind("KubernetesImagePuller")),
			},
		},
	}
}
