//
// Copyright (c) 2019-2020 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package solver

import (
	controllerv1alpha1 "github.com/devfile/devworkspace-operator/apis/controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/controllers/controller/workspacerouting/solvers"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger = ctrl.Log.WithName("solver")
)

// CheRoutingSolver is a struct representing the routing solver for Che specific routing of workspaces
type CheRoutingSolver struct {
	client client.Client
	scheme *runtime.Scheme
}

// New creates a new Che routing solver
func New(client client.Client, scheme *runtime.Scheme) *CheRoutingSolver {
	return &CheRoutingSolver{
		client: client,
		scheme: scheme,
	}
}

// GetSpecObjects constructs cluster routing objects which should be applied on the cluster
func (c *CheRoutingSolver) GetSpecObjects(spec controllerv1alpha1.WorkspaceRoutingSpec, workspaceMeta solvers.WorkspaceMetadata) solvers.RoutingObjects {
	// TODO specify what services in singlehost or service-ingress/route pairs in multihost need to be created
	// in singlehost mode we additionally need configmaps in the namespace of the Che CR
	return solvers.RoutingObjects{}
}

// GetExposedEndpoints retreives the URL for each endpoint in a devfile spec from a set of RoutingObjects.
// Returns is a map from component ids (as defined in the devfile) to the list of endpoints for that component
// Return value "ready" specifies if all endpoints are resolved on the cluster; if false it is necessary to retry, as
// URLs will be undefined.
func (c *CheRoutingSolver) GetExposedEndpoints(endpoints map[string]controllerv1alpha1.EndpointList, routingObj solvers.RoutingObjects) (exposedEndpoints map[string]controllerv1alpha1.ExposedEndpointList, ready bool, err error) {
	// TODO implement this
	return nil, false, nil
}
