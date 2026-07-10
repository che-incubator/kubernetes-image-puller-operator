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
	"fmt"

	chev1alpha1 "github.com/che-incubator/kubernetes-image-puller-operator/api/v1alpha1"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/config"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/defaults"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/rbac"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	log := r.Log.WithValues("kubernetesimagepuller", req.NamespacedName)

	// Fetch the KubernetesImagePuller instance
	instance := &chev1alpha1.KubernetesImagePuller{}
	err := r.Get(ctx, req.NamespacedName, instance)
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

	// Set defaults for any unset spec fields in a single update
	needsUpdate := false
	if instance.Spec.ConfigMapName == "" {
		instance.Spec.ConfigMapName = defaults.ConfigMapName
		needsUpdate = true
	}
	if instance.Spec.DeploymentName == "" {
		instance.Spec.DeploymentName = defaults.DeploymentName
		needsUpdate = true
	}
	if instance.Spec.ImagePullerImage == "" {
		instance.Spec.ImagePullerImage = defaults.ImagePullerImage
		needsUpdate = true
	}
	if needsUpdate {
		if err = r.Update(ctx, instance); err != nil {
			log.Error(err, "Error updating KubernetesImagePuller")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create the Role to allow the ServiceAccount to create Daemonsets
	foundRole := &rbacv1.Role{}
	err = r.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: defaults.RBACName}, foundRole)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating create-daemonset role")
		if err = r.Create(ctx, rbac.NewRole(instance)); err != nil {
			log.Error(err, "Error creating create-daemonset role")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	foundRoleBinding := &rbacv1.RoleBinding{}
	err = r.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: defaults.RBACName}, foundRoleBinding)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating create-daemonset RoleBinding")
		if err = r.Create(ctx, rbac.NewRoleBinding(instance)); err != nil {
			log.Error(err, "Error creating create-daemonset RoleBinding")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	foundServiceAccount := &corev1.ServiceAccount{}
	err = r.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: defaults.ServiceAccountName}, foundServiceAccount)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating k8s-image-puller ServiceAccount")
		if err = r.Create(ctx, rbac.NewServiceAccount(instance)); err != nil {
			log.Error(err, "Error creating k8s-image-puller ServiceAccount")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Create the configmap if it does not exist
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Spec.ConfigMapName, Namespace: instance.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating kubernetes image puller ConfigMap", "ConfigMap.Namespace", instance.Namespace)
		err = r.Create(ctx, config.NewImagePullerConfigMap(instance))
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// If there is an existing deployment, roll it out on configmap change
	oldDeployment := &appsv1.Deployment{}
	if err = r.Get(ctx, types.NamespacedName{Name: instance.Spec.DeploymentName, Namespace: instance.Namespace}, oldDeployment); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Error getting deployment")
		return ctrl.Result{}, err
	} else if err == nil {
		oldConfigMapName, err := GetDeploymentConfigMapName(oldDeployment)
		if err != nil {
			log.Error(err, "Error reading configmap name from deployment")
			return ctrl.Result{}, err
		}
		if config.ConfigMapsDiffer(config.NewImagePullerConfigMap(instance), foundConfigMap) || oldConfigMapName != foundConfigMap.Name {
			if oldConfigMapName == foundConfigMap.Name {
				// ConfigMap names are the same, delete pods and let the new pods pick up the new configmap

				pods := &corev1.PodList{}
				if err = r.List(ctx, pods, client.MatchingLabels{"app": defaults.AppLabelValue}); err != nil {
					log.Error(err, "Error listing pods")
					return ctrl.Result{}, err
				}
				if len(pods.Items) > 0 {
					for _, pod := range pods.Items {
						log.Info("Deleting pod", "Pod.Name", pod.Name)
						if err = r.Delete(ctx, &pod); err != nil {
							log.Error(err, "Error deleting pod", "Pod.Name", pod.Name)
							return ctrl.Result{}, err
						}
					}
				}
				// ConfigMap names are different, just run an update
			} else {
				err = r.Update(ctx, NewImagePullerDeployment(instance))
				if err != nil {
					log.Error(err, "Error updating deployment")
					return ctrl.Result{}, err
				}
			}
		}
	}

	// If the configmap has already been created and the values have changed, update the configmap.
	newConfigMap := config.NewImagePullerConfigMap(instance)
	if config.ConfigMapsDiffer(newConfigMap, foundConfigMap) {
		if err = r.Update(ctx, newConfigMap); err != nil {
			log.Error(err, "Error updating configmap")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// Find all configmaps owned by this imagepuller and delete ones that are not named ConfigMapName
	configMaps := &corev1.ConfigMapList{}
	if err = r.List(ctx, configMaps, client.MatchingLabels{"app": defaults.AppLabelValue}); err != nil {
		log.Error(err, "Could not list ConfigMaps")
		return ctrl.Result{}, err
	} else {

		for _, cm := range configMaps.Items {
			if cm.Name != instance.Spec.ConfigMapName && len(cm.ObjectMeta.OwnerReferences) > 0 {
				if cm.ObjectMeta.OwnerReferences[0].Name == instance.Name {
					if err = r.Delete(ctx, &cm); err != nil {
						log.Error(err, "Could not delete ConfigMap")
						return ctrl.Result{}, err
					}
				}
			}
		}
	}

	// Create a new ConfigMap if the names differ
	if foundConfigMap.Name != instance.Spec.ConfigMapName {
		newConfigMap := config.NewImagePullerConfigMap(instance)
		if err = r.Create(ctx, newConfigMap); err != nil {
			log.Error(err, "Could not create new ConfigMap")
			return ctrl.Result{}, err
		}
		if err = r.Delete(ctx, foundConfigMap); err != nil {
			log.Error(err, "Could not delete old ConfigMap")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Check if kubernetes image puller deployment exists, and create it if it does not.
	foundDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Spec.DeploymentName, Namespace: instance.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating kubernetes image puller deployment", "Deployment.Namespace", instance.Namespace)
		err = r.Create(ctx, NewImagePullerDeployment(instance))
		if err != nil {
			log.Error(err, "Could not create deployment")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "Could not get deployment")
		return ctrl.Result{}, err
	}

	// If DeploymentName has changed, delete the old deployment and create a new one
	deployments := &appsv1.DeploymentList{}
	listOptions := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labels.SelectorFromValidatedSet(map[string]string{"app": defaults.AppLabelValue}),
	}
	err = r.List(ctx, deployments, listOptions)
	if err != nil {
		log.Error(err, "Error listing deployments")
		return ctrl.Result{}, err
	}
	// If more than one deployment found in list, delete all deployments not named instance.Spec.DeploymentName
	if len(deployments.Items) > 1 {
		for _, deployment := range deployments.Items {
			if deployment.Name != instance.Spec.DeploymentName {
				log.Info("Deleting old deployment", "Deployment.Name", deployment.Name)
				err = r.Delete(ctx, &deployment)
				if err != nil {
					log.Error(err, "Could not delete deployment")
					return ctrl.Result{}, err
				}
			}
		}
	}

	// If ImagePullerImage from deployment is different than the spec, update deployment
	if instance.Spec.ImagePullerImage != instance.Status.ImagePullerImage {
		instance.Status.ImagePullerImage = instance.Spec.ImagePullerImage
		err = r.Status().Update(ctx, instance)
		if err != nil {
			log.Error(err, "Error updating custom resource status")
			return ctrl.Result{}, err
		}

		err = r.Update(ctx, NewImagePullerDeployment(instance))
		if err != nil {
			log.Error(err, "Error updating deployment")
			return ctrl.Result{}, err
		}
	}

	// Everything already exists
	log.Info("End Reconcile")

	return ctrl.Result{}, nil
}

func NewImagePullerDeployment(cr *chev1alpha1.KubernetesImagePuller) *appsv1.Deployment {
	replicas := int32(1)
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := true
	var deploymentName string
	if cr.Spec.DeploymentName == "" {
		deploymentName = defaults.DeploymentName
	} else {
		deploymentName = cr.Spec.DeploymentName
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      deploymentName,
			Labels:    map[string]string{"app": defaults.AppLabelValue},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, chev1alpha1.SchemeBuilder.GroupVersion.WithKind("KubernetesImagePuller")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": defaults.AppLabelValue},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": defaults.AppLabelValue},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: defaults.ServiceAccountName,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runAsNonRoot,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  defaults.ContainerName,
							Image: cr.Spec.ImagePullerImage,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
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

func GetDeploymentConfigMapName(deployment *appsv1.Deployment) (string, error) {
	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		return "", fmt.Errorf("deployment %s/%s has no containers", deployment.Namespace, deployment.Name)
	}
	envFrom := containers[0].EnvFrom
	if len(envFrom) == 0 || envFrom[0].ConfigMapRef == nil {
		return "", fmt.Errorf("deployment %s/%s has no configmap env source", deployment.Namespace, deployment.Name)
	}
	return envFrom[0].ConfigMapRef.Name, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubernetesImagePullerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chev1alpha1.KubernetesImagePuller{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
