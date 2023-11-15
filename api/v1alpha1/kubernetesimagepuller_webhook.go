//
// Copyright (c) 2012-2023 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var kubernetesimagepullerlog = logf.Log.WithName("kubernetesimagepuller-resource")

func (r *KubernetesImagePuller) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-che-eclipse-org-v1alpha1-kubernetesimagepuller,mutating=true,failurePolicy=fail,sideEffects=None,groups=che.eclipse.org,resources=kubernetesimagepullers,verbs=create;update,versions=v1alpha1,name=mkubernetesimagepuller.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &KubernetesImagePuller{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *KubernetesImagePuller) Default() {
	kubernetesimagepullerlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-che-eclipse-org-v1alpha1-kubernetesimagepuller,mutating=false,failurePolicy=fail,sideEffects=None,groups=che.eclipse.org,resources=kubernetesimagepullers,verbs=create;update,versions=v1alpha1,name=vkubernetesimagepuller.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &KubernetesImagePuller{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *KubernetesImagePuller) ValidateCreate() error {
	kubernetesimagepullerlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *KubernetesImagePuller) ValidateUpdate(old runtime.Object) error {
	kubernetesimagepullerlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *KubernetesImagePuller) ValidateDelete() error {
	kubernetesimagepullerlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
