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

package defaults

const (
	ConfigMapName    = "k8s-image-puller"
	DeploymentName   = "kubernetes-image-puller"
	ImagePullerImage = "quay.io/eclipse/kubernetes-image-puller:next"

	AppLabelValue      = "kubernetes-image-puller"
	ContainerName      = "kubernetes-image-puller"
	DaemonSetName      = "kubernetes-image-puller"
	RBACName           = "create-daemonset"
	ServiceAccountName = "k8s-image-puller"
)
