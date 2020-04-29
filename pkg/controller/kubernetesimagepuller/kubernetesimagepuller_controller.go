package kubernetesimagepuller

import (
	"context"

	chev1alpha1 "github.com/che-incubator/kubernetes-image-puller-operator/pkg/apis/che/v1alpha1"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/config"
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/rbac"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_kubernetesimagepuller")

// Add creates a new KubernetesImagePuller Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKubernetesImagePuller{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kubernetesimagepuller-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KubernetesImagePuller
	err = c.Watch(&source.Kind{Type: &chev1alpha1.KubernetesImagePuller{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch DaemonSets
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &chev1alpha1.KubernetesImagePuller{},
	})
	if err != nil {
		return err
	}

	// Watch Deployments
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &chev1alpha1.KubernetesImagePuller{},
	})
	if err != nil {
		return err
	}

	// Watch Pods
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &chev1alpha1.KubernetesImagePuller{},
	})

	// Watch ConfigMaps
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &chev1alpha1.KubernetesImagePuller{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileKubernetesImagePuller{}

// ReconcileKubernetesImagePuller reconciles a KubernetesImagePuller object
type ReconcileKubernetesImagePuller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KubernetesImagePuller object and makes changes based on the state read
// and what is in the KubernetesImagePuller.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKubernetesImagePuller) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KubernetesImagePuller")

	// Fetch the KubernetesImagePuller instance
	instance := &chev1alpha1.KubernetesImagePuller{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// If there is no set configmap name, update with the default configmap name
	if instance.Spec.ConfigMapName == "" {
		instance.Spec.ConfigMapName = "k8s-image-puller"
		if err = r.client.Update(context.TODO(), instance); err != nil {
			reqLogger.Error(err, "Error updating KubernetesImagePuller")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// If there is no set deployment name, update with the default deployment name
	if instance.Spec.DeploymentName == "" {
		instance.Spec.DeploymentName = "kubernetes-image-puller"
		if err = r.client.Update(context.TODO(), instance); err != nil {
			reqLogger.Error(err, "Error updating KubernetesImagePuller")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Create the Role to allow the ServiceAccount to create Daemonsets
	foundRole := &rbacv1.Role{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: "create-daemonset"}, foundRole)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating create-daemonset role")
		if err = r.client.Create(context.TODO(), rbac.NewRole(instance)); err != nil {
			reqLogger.Error(err, "Error creating create-daemonset role")
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	foundRoleBinding := &rbacv1.RoleBinding{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: "create-daemonset"}, foundRoleBinding)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating create-daemonset RoleBinding")
		if err = r.client.Create(context.TODO(), rbac.NewRoleBinding(instance)); err != nil {
			reqLogger.Error(err, "Error creating create-daemonset RoleBinding")
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	foundServiceAccount := &corev1.ServiceAccount{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: "k8s-image-puller"}, foundServiceAccount)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating k8s-image-puller ServiceAccount")
		if err = r.client.Create(context.TODO(), rbac.NewServiceAccount(instance)); err != nil {
			reqLogger.Error(err, "Error creating k8s-image-puller ServiceAccount")
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Create the configmap if it does not exist
	foundConfigMap := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ConfigMapName, Namespace: instance.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating kubernetes image puller ConfigMap", "ConfigMap.Namespace", instance.Namespace)
		err = r.client.Create(context.TODO(), config.NewImagePullerConfigMap(instance))
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// If there is an existing deployment, roll it out on configmap change
	oldDeployment := &appsv1.Deployment{}
	if err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.DeploymentName, Namespace: instance.Namespace}, oldDeployment); err != nil && !errors.IsNotFound(err) {
		reqLogger.Error(err, "Error getting deployment")
		return reconcile.Result{}, err
	} else if err == nil {
		if config.ConfigMapsDiffer(config.NewImagePullerConfigMap(instance), foundConfigMap) || GetDeploymentConfigMapName(oldDeployment) != foundConfigMap.Name {
			if GetDeploymentConfigMapName(oldDeployment) == foundConfigMap.Name {
				// ConfigMap names are the same, delete pods and let the new pods pick up the new configmap

				pods := &corev1.PodList{}
				if err = r.client.List(context.TODO(), pods, client.MatchingLabels{"app": "kubernetes-image-puller"}); err != nil {
					reqLogger.Error(err, "Error listing pods")
					return reconcile.Result{}, err
				}
				if len(pods.Items) > 0 {
					for _, pod := range pods.Items {
						reqLogger.Info("Deleting pod", "Pod.Name", pod.Name)
						if err = r.client.Delete(context.TODO(), &pod); err != nil {
							reqLogger.Error(err, "Error deleting pod", "Pod.Name", pod.Name)
							return reconcile.Result{}, err
						}
					}
				}
				// ConfigMap names are different, just run an update
			} else {
				err = r.client.Update(context.TODO(), NewImagePullerDeployment(instance))
				if err != nil {
					reqLogger.Error(err, "Error updating deployment")
					return reconcile.Result{}, err
				}
			}
		}
	}

	// If the configmap has already been created and the values have changed, update the configmap.
	newConfigMap := config.NewImagePullerConfigMap(instance)
	if config.ConfigMapsDiffer(newConfigMap, foundConfigMap) {
		if err = r.client.Update(context.TODO(), newConfigMap); err != nil {
			reqLogger.Error(err, "Error updating configmap")
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	// Find all configmaps owned by this imagepuller and delete ones that are not named ConfigMapName
	configMaps := &corev1.ConfigMapList{}
	if err = r.client.List(context.TODO(), configMaps, client.MatchingLabels{"app": "kubernetes-image-puller"}); err != nil {
		reqLogger.Error(err, "Could not list ConfigMaps")
		return reconcile.Result{}, err
	} else {

		for _, cm := range configMaps.Items {
			if cm.Name != instance.Spec.ConfigMapName && len(cm.ObjectMeta.OwnerReferences) > 0 {
				if cm.ObjectMeta.OwnerReferences[0].Name == instance.Name {
					if err = r.client.Delete(context.TODO(), &cm); err != nil {
						reqLogger.Error(err, "Could not delete ConfigMap")
						return reconcile.Result{}, err
					}
				}
			}
		}
	}

	// Create a new ConfigMap if the names differ
	if foundConfigMap.Name != instance.Spec.ConfigMapName {
		newConfigMap := config.NewImagePullerConfigMap(instance)
		if err = r.client.Create(context.TODO(), newConfigMap); err != nil {
			reqLogger.Error(err, "Could not create new ConfigMap")
			return reconcile.Result{}, err
		}
		if err = r.client.Delete(context.TODO(), foundConfigMap); err != nil {
			reqLogger.Error(err, "Could not delete old ConfigMap")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Check if kubernetes image puller deployment exists, and create it if it does not.
	foundDeployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.DeploymentName, Namespace: instance.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating kubernetes image puller deployment", "Deployment.Namespace", instance.Namespace)
		err = r.client.Create(context.TODO(), NewImagePullerDeployment(instance))
		if err != nil {
			reqLogger.Error(err, "Could not create deployment")
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		reqLogger.Error(err, "Could not get deployment")
		return reconcile.Result{}, err
	}

	// If DeploymentName has changed, delete the old deployment and create a new one
	deployments := &appsv1.DeploymentList{}
	err = r.client.List(context.TODO(), deployments, client.MatchingLabels{"app": "kubernetes-image-puller"})
	if err != nil {
		reqLogger.Error(err, "Error listing deployments")
		return reconcile.Result{}, err
	}
	// If more than one deployment found in list, delete all deployments not named instance.Spec.DeploymentName
	if len(deployments.Items) > 1 {
		for _, deployment := range deployments.Items {
			if deployment.Name != instance.Spec.DeploymentName {
				reqLogger.Info("Deleting old deployment", "Deployment.Name", deployment.Name)
				err = r.client.Delete(context.TODO(), &deployment)
				if err != nil {
					reqLogger.Error(err, "Could not delete deployment")
					return reconcile.Result{}, err
				}
			}
		}
	}

	// Everything already exists
	reqLogger.Info("End Reconcile")
	return reconcile.Result{}, nil
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
				*metav1.NewControllerRef(cr, chev1alpha1.SchemeGroupVersion.WithKind("KubernetesImagePuller")),
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
							Image: "quay.io/eclipse/kubernetes-image-puller:latest",
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

// Used for testing, adds the controller to a manager
func (r *ReconcileKubernetesImagePuller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chev1alpha1.KubernetesImagePuller{}).
		Complete(r)
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *chev1alpha1.KubernetesImagePuller) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
