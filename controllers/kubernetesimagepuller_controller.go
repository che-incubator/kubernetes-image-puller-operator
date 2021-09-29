/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	chev1alpha1 "github.com/che-incubator/kubernetes-image-puller-operator/api/v1alpha1"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/config"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/rbac"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// KubernetesImagePullerReconciler reconciles a KubernetesImagePuller object
type KubernetesImagePullerReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KubernetesImagePuller object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *KubernetesImagePullerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("kubernetesimagepuller", req.NamespacedName)

	// Fetch the KubernetesImagePuller instance
	instance := &chev1alpha1.KubernetesImagePuller{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// If there is no set configmap name, update with the default configmap name
	if instance.Spec.ConfigMapName == "" {
		instance.Spec.ConfigMapName = "k8s-image-puller"
		if err = r.Update(context.TODO(), instance); err != nil {
			r.Log.Error(err, "Error updating KubernetesImagePuller")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If there is no set deployment name, update with the default deployment name
	if instance.Spec.DeploymentName == "" {
		instance.Spec.DeploymentName = "kubernetes-image-puller"
		if err = r.Update(context.TODO(), instance); err != nil {
			r.Log.Error(err, "Error updating KubernetesImagePuller")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If there is no image puller image set, update with the default image puller image
	if instance.Spec.ImagePullerImage == "" {
		instance.Spec.ImagePullerImage = "quay.io/eclipse/kubernetes-image-puller:next"
		if err = r.Update(context.TODO(), instance); err != nil {
			r.Log.Error(err, "Error updating KubernetesImagePuller")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create the Role to allow the ServiceAccount to create Daemonsets
	foundRole := &rbacv1.Role{}
	err = r.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: "create-daemonset"}, foundRole)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating create-daemonset role")
		if err = r.Create(context.TODO(), rbac.NewRole(instance)); err != nil {
			r.Log.Error(err, "Error creating create-daemonset role")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	foundRoleBinding := &rbacv1.RoleBinding{}
	err = r.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: "create-daemonset"}, foundRoleBinding)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating create-daemonset RoleBinding")
		if err = r.Create(context.TODO(), rbac.NewRoleBinding(instance)); err != nil {
			r.Log.Error(err, "Error creating create-daemonset RoleBinding")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	foundServiceAccount := &corev1.ServiceAccount{}
	err = r.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: "k8s-image-puller"}, foundServiceAccount)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating k8s-image-puller ServiceAccount")
		if err = r.Create(context.TODO(), rbac.NewServiceAccount(instance)); err != nil {
			r.Log.Error(err, "Error creating k8s-image-puller ServiceAccount")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Create the configmap if it does not exist
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ConfigMapName, Namespace: instance.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating kubernetes image puller ConfigMap", "ConfigMap.Namespace", instance.Namespace)
		err = r.Create(context.TODO(), config.NewImagePullerConfigMap(instance))
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// If there is an existing deployment, roll it out on configmap change
	oldDeployment := &appsv1.Deployment{}
	if err = r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.DeploymentName, Namespace: instance.Namespace}, oldDeployment); err != nil && !errors.IsNotFound(err) {
		r.Log.Error(err, "Error getting deployment")
		return ctrl.Result{}, err
	} else if err == nil {
		if config.ConfigMapsDiffer(config.NewImagePullerConfigMap(instance), foundConfigMap) || GetDeploymentConfigMapName(oldDeployment) != foundConfigMap.Name {
			if GetDeploymentConfigMapName(oldDeployment) == foundConfigMap.Name {
				// ConfigMap names are the same, delete pods and let the new pods pick up the new configmap

				pods := &corev1.PodList{}
				if err = r.List(context.TODO(), pods, client.MatchingLabels{"app": "kubernetes-image-puller"}); err != nil {
					r.Log.Error(err, "Error listing pods")
					return ctrl.Result{}, err
				}
				if len(pods.Items) > 0 {
					for _, pod := range pods.Items {
						r.Log.Info("Deleting pod", "Pod.Name", pod.Name)
						if err = r.Delete(context.TODO(), &pod); err != nil {
							r.Log.Error(err, "Error deleting pod", "Pod.Name", pod.Name)
							return ctrl.Result{}, err
						}
					}
				}
				// ConfigMap names are different, just run an update
			} else {
				err = r.Update(context.TODO(), NewImagePullerDeployment(instance))
				if err != nil {
					r.Log.Error(err, "Error updating deployment")
					return ctrl.Result{}, err
				}
			}
		}
	}

	// If the configmap has already been created and the values have changed, update the configmap.
	newConfigMap := config.NewImagePullerConfigMap(instance)
	if config.ConfigMapsDiffer(newConfigMap, foundConfigMap) {
		if err = r.Update(context.TODO(), newConfigMap); err != nil {
			r.Log.Error(err, "Error updating configmap")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// Find all configmaps owned by this imagepuller and delete ones that are not named ConfigMapName
	configMaps := &corev1.ConfigMapList{}
	if err = r.List(context.TODO(), configMaps, client.MatchingLabels{"app": "kubernetes-image-puller"}); err != nil {
		r.Log.Error(err, "Could not list ConfigMaps")
		return ctrl.Result{}, err
	} else {

		for _, cm := range configMaps.Items {
			if cm.Name != instance.Spec.ConfigMapName && len(cm.ObjectMeta.OwnerReferences) > 0 {
				if cm.ObjectMeta.OwnerReferences[0].Name == instance.Name {
					if err = r.Delete(context.TODO(), &cm); err != nil {
						r.Log.Error(err, "Could not delete ConfigMap")
						return ctrl.Result{}, err
					}
				}
			}
		}
	}

	// Create a new ConfigMap if the names differ
	if foundConfigMap.Name != instance.Spec.ConfigMapName {
		newConfigMap := config.NewImagePullerConfigMap(instance)
		if err = r.Create(context.TODO(), newConfigMap); err != nil {
			r.Log.Error(err, "Could not create new ConfigMap")
			return ctrl.Result{}, err
		}
		if err = r.Delete(context.TODO(), foundConfigMap); err != nil {
			r.Log.Error(err, "Could not delete old ConfigMap")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Check if kubernetes image puller deployment exists, and create it if it does not.
	foundDeployment := &appsv1.Deployment{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.DeploymentName, Namespace: instance.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating kubernetes image puller deployment", "Deployment.Namespace", instance.Namespace)
		err = r.Create(context.TODO(), NewImagePullerDeployment(instance))
		if err != nil {
			r.Log.Error(err, "Could not create deployment")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	} else if err != nil {
		r.Log.Error(err, "Could not get deployment")
		return ctrl.Result{}, err
	}

	// If DeploymentName has changed, delete the old deployment and create a new one
	deployments := &appsv1.DeploymentList{}
	err = r.List(context.TODO(), deployments, client.MatchingLabels{"app": "kubernetes-image-puller"})
	if err != nil {
		r.Log.Error(err, "Error listing deployments")
		return ctrl.Result{}, err
	}
	// If more than one deployment found in list, delete all deployments not named instance.Spec.DeploymentName
	if len(deployments.Items) > 1 {
		for _, deployment := range deployments.Items {
			if deployment.Name != instance.Spec.DeploymentName {
				r.Log.Info("Deleting old deployment", "Deployment.Name", deployment.Name)
				err = r.Delete(context.TODO(), &deployment)
				if err != nil {
					r.Log.Error(err, "Could not delete deployment")
					return ctrl.Result{}, err
				}
			}
		}
	}

	// If ImagePullerImage from deployment is different than the spec, update deployment
	if instance.Spec.ImagePullerImage != instance.Status.ImagePullerImage {
		instance.Status.ImagePullerImage = instance.Spec.ImagePullerImage
		err = r.Status().Update(context.TODO(), instance)
		if err != nil {
			r.Log.Error(err, "Error updating custom resource status")
			return ctrl.Result{}, err
		}

		err = r.Update(context.TODO(), NewImagePullerDeployment(instance))
		if err != nil {
			r.Log.Error(err, "Error updating deployment")
			return ctrl.Result{}, err
		}
	}

	// Everything already exists
	r.Log.Info("End Reconcile")

	return ctrl.Result{}, nil
}

func NewImagePullerDeployment(cr *chev1alpha1.KubernetesImagePuller) *appsv1.Deployment {
	replicas := int32(1)
	var deploymentName string
	if cr.Spec.DeploymentName == "" {
		deploymentName = "kubernetes-image-puller"
	} else {
		deploymentName = cr.Spec.DeploymentName
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      deploymentName,
			Labels:    map[string]string{"app": "kubernetes-image-puller"},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, chev1alpha1.SchemeBuilder.GroupVersion.WithKind("KubernetesImagePuller")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "kubernetes-image-puller"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "kubernetes-image-puller"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "k8s-image-puller",
					Containers: []corev1.Container{
						{
							Name:  "kubernetes-image-puller",
							Image: cr.Spec.ImagePullerImage,
							Env: []corev1.EnvVar{{
								Name:  "DEPLOYMENT_NAME",
								Value: deploymentName,
							}},
							EnvFrom: []corev1.EnvFromSource{{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cr.Spec.ConfigMapName,
									},
								},
							}},
						},
					},
				},
			},
		},
	}
}

// Returns the name of the configmap used as an environment source
func GetDeploymentConfigMapName(deployment *appsv1.Deployment) string {
	return deployment.Spec.Template.Spec.Containers[0].EnvFrom[0].ConfigMapRef.Name
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubernetesImagePullerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chev1alpha1.KubernetesImagePuller{}).
		Watches(&source.Kind{Type: &chev1alpha1.KubernetesImagePuller{}}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &chev1alpha1.KubernetesImagePuller{},
		}).
		Watches(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &chev1alpha1.KubernetesImagePuller{},
		}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &chev1alpha1.KubernetesImagePuller{},
		}).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &chev1alpha1.KubernetesImagePuller{},
		}).
		Complete(r)
}
