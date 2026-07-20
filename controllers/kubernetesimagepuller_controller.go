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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

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
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	legacyImageV111 = "quay.io/eclipse/kubernetes-image-puller:1.1.1"
	legacyImageNext = "quay.io/eclipse/kubernetes-image-puller:next"
)

// KubernetesImagePullerReconciler reconciles a KubernetesImagePuller object
type KubernetesImagePullerReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	IsOpenShift bool
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

	if instance.Spec.ImagePullerImage == legacyImageV111 || instance.Spec.ImagePullerImage == legacyImageNext {
		instance.Spec.ImagePullerImage = ""
		needsUpdate = true
	}

	if needsUpdate {
		if err = r.Update(ctx, instance); err != nil {
			log.Error(err, "Error updating KubernetesImagePuller")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create or update the Role to allow the ServiceAccount to create Daemonsets
	foundRole := &rbacv1.Role{}
	err = r.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: defaults.RBACName}, foundRole)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating create-daemonset role")
		if err = r.Create(ctx, rbac.NewRole(instance)); err != nil {
			log.Error(err, "Error creating create-daemonset role")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		desiredRole := rbac.NewRole(instance)
		if !reflect.DeepEqual(foundRole.Rules, desiredRole.Rules) {
			foundRole.Rules = desiredRole.Rules
			if err = r.Update(ctx, foundRole); err != nil {
				log.Error(err, "Error updating create-daemonset role")
				return ctrl.Result{}, err
			}
		}
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
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		desiredRoleBinding := rbac.NewRoleBinding(instance)
		if !reflect.DeepEqual(foundRoleBinding.RoleRef, desiredRoleBinding.RoleRef) {
			// RoleRef is immutable — delete and let the next reconcile recreate it
			if err = r.Delete(ctx, foundRoleBinding); err != nil {
				log.Error(err, "Error deleting create-daemonset RoleBinding for roleRef update")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		if !reflect.DeepEqual(foundRoleBinding.Subjects, desiredRoleBinding.Subjects) {
			foundRoleBinding.Subjects = desiredRoleBinding.Subjects
			if err = r.Update(ctx, foundRoleBinding); err != nil {
				log.Error(err, "Error updating create-daemonset RoleBinding")
				return ctrl.Result{}, err
			}
		}
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

	// Update the ConfigMap before triggering a rollout so new pods
	// always read the persisted data.
	newConfigMap := config.NewImagePullerConfigMap(instance)
	configMapUpdated := false
	if config.ConfigMapsDiffer(newConfigMap, foundConfigMap) {
		if err = r.Update(context.TODO(), newConfigMap); err != nil {
			r.Log.Error(err, "Error updating configmap")
			return ctrl.Result{}, err
		}
		configMapUpdated = true
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
		if configMapUpdated || oldConfigMapName != foundConfigMap.Name {
			if oldConfigMapName == foundConfigMap.Name {
				desiredHash := configMapDataHash(newConfigMap.Data)
				if oldDeployment.Spec.Template.Annotations == nil {
					oldDeployment.Spec.Template.Annotations = make(map[string]string)
				}
				oldDeployment.Spec.Template.Annotations["che.eclipse.org/configmap-hash"] = desiredHash
				log.Info("ConfigMap content changed, triggering rolling restart")
				if err = r.Update(ctx, oldDeployment); err != nil {
					log.Error(err, "Error updating deployment for rollout")
					return ctrl.Result{}, err
				}
			} else {
				err = r.Update(ctx, NewImagePullerDeployment(instance, r.IsOpenShift))
				if err != nil {
					log.Error(err, "Error updating deployment")
					return ctrl.Result{}, err
				}
			}
		}
	}

	if configMapUpdated {
		return ctrl.Result{Requeue: true}, nil
	}

	// Find all configmaps owned by this imagepuller and delete ones that are not named ConfigMapName
	configMaps := &corev1.ConfigMapList{}
	if err = r.List(ctx, configMaps, client.MatchingLabels{"app": defaults.AppLabelValue}); err != nil {
		log.Error(err, "Could not list ConfigMaps")
		return ctrl.Result{}, err
	} else {

		for _, cm := range configMaps.Items {
			if cm.Name != instance.Spec.ConfigMapName && len(cm.OwnerReferences) > 0 {
				if cm.OwnerReferences[0].Name == instance.Name {
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
		err = r.Create(ctx, NewImagePullerDeployment(instance, r.IsOpenShift))
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
	effectiveImage := instance.GetImagePullerImage()
	if instance.Status.ImagePullerImage != effectiveImage {
		instance.Status.ImagePullerImage = effectiveImage
		err = r.Status().Update(ctx, instance)
		if err != nil {
			log.Error(err, "Error updating custom resource status")
			return ctrl.Result{}, err
		}

		err = r.Update(ctx, NewImagePullerDeployment(instance, r.IsOpenShift))
		if err != nil {
			log.Error(err, "Error updating deployment")
			return ctrl.Result{}, err
		}
	}

	// Everything already exists
	log.Info("End Reconcile")

	return ctrl.Result{}, nil
}

func NewImagePullerDeployment(cr *chev1alpha1.KubernetesImagePuller, isOpenShift bool) *appsv1.Deployment {
	replicas := int32(1)

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
				*metav1.NewControllerRef(cr, chev1alpha1.GroupVersion.WithKind("KubernetesImagePuller")),
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
					SecurityContext:    getPodSecurityContext(isOpenShift),
					Containers: []corev1.Container{
						{
							Name:            defaults.ContainerName,
							Image:           cr.GetImagePullerImage(),
							SecurityContext: getContainerSecurityContext(isOpenShift),
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

func getPodSecurityContext(isOpenShift bool) *corev1.PodSecurityContext {
	securityContext := &corev1.PodSecurityContext{
		RunAsNonRoot:   ptr.To(true),
		SeccompProfile: ptr.To(defaults.SeccompProfile),
	}

	if !isOpenShift {
		securityContext.FSGroup = ptr.To(defaults.NonRootGID)
	}

	return securityContext
}

func getContainerSecurityContext(isOpenShift bool) *corev1.SecurityContext {
	ctx := &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		ReadOnlyRootFilesystem:   ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
	}

	if !isOpenShift {
		ctx.RunAsUser = ptr.To(defaults.NonRootUID)
		ctx.RunAsGroup = ptr.To(defaults.NonRootGID)
	}

	return ctx
}

func configMapDataHash(data map[string]string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	raw, _ := json.Marshal(keys)
	for _, k := range keys {
		v, _ := json.Marshal(data[k])
		raw = append(raw, v...)
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubernetesImagePullerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chev1alpha1.KubernetesImagePuller{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
