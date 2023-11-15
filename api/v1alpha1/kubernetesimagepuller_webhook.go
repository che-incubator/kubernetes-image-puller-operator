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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	authorizationv1 "k8s.io/api/authorization/v1"
	authorizationv1Client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (r *KubernetesImagePuller) SetupWebhookWithManager(mgr ctrl.Manager) error {

	authClient, err := authorizationv1Client.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}

	mgr.GetWebhookServer().Register("/validate-che-eclipse-org-v1alpha1-kubernetesimagepuller", &webhook.Admission{Handler: &validationHandler{subjectAccessReviews: authClient.SubjectAccessReviews()}})
	return nil
}

// +kubebuilder:webhook:path=/validate-che-eclipse-org-v1alpha1-kubernetesimagepuller,mutating=false,failurePolicy=fail,sideEffects=None,groups=che.eclipse.org,resources=kubernetesimagepullers,verbs=create,versions=v1alpha1,name=vkubernetesimagepuller.kb.io,admissionReviewVersions={v1,v1beta1}

type validationHandler struct {
	subjectAccessReviews authorizationv1Client.SubjectAccessReviewInterface
}

func (v *validationHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: req.Namespace,
		Group:     "apps",
		Version:   "*",
		Verb:      "create",
		Resource:  "daemonsets",
	}

	subjectAccessReview := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			ResourceAttributes: resourceAttributes,
			User:               req.UserInfo.Username,
			UID:                req.UserInfo.UID,
			Groups:             req.UserInfo.Groups,
		},
	}

	resp, _ := v.subjectAccessReviews.Create(ctx, subjectAccessReview, metav1.CreateOptions{})

	if !resp.Status.Allowed {
		return webhook.Denied("User \"" + req.UserInfo.Username + "\" not allowed to create daemonsets")
	}

	return webhook.Allowed("User \"" + req.UserInfo.Username + "\" allowed to create daemonsets")
}
