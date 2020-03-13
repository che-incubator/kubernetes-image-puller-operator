package controller

import (
	"github.com/che-incubator/kubernetes-image-puller-operator/pkg/controller/kubernetesimagepuller"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, kubernetesimagepuller.Add)
}
