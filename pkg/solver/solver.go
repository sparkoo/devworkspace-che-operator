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
	client.Client
	scheme *runtime.Scheme
}

// CheRouterGetter negotiates the solver with the calling code
type CheRouterGetter struct {
	scheme *runtime.Scheme
}

// Getter creates a new CheRouterGetter
func Getter(scheme *runtime.Scheme) *CheRouterGetter {
	return &CheRouterGetter{
		scheme: scheme,
	}
}

// HasSolver returns whether the provided routingClass is supported by this RoutingSolverGetter. Returns false if
// calling GetSolver with routingClass will return a RoutingNotSupported error. Can be used to check if a routingClass
// is supported without having to provide a runtime client. Note that GetSolver may still return another error, if e.g.
// an OpenShift-only routingClass is used on a vanilla Kubernetes platform.
func (g *CheRouterGetter) HasSolver(routingClass controllerv1alpha1.WorkspaceRoutingClass) bool {
	return isSupported(routingClass)
}

// GetSolver that obtains a Solver (see github.com/devfile/devworkspace-operator/controllers/controller/workspacerouting/solvers)
// for a particular WorkspaceRouting instance. This function should return a RoutingNotSupported error if
// the routingClass is not recognized, and any other error if the routingClass is invalid (e.g. an OpenShift-only
// routingClass on a vanilla Kubernetes platform). Note that an empty routingClass is handled by the DevWorkspace controller itself,
// and should not be handled by external controllers.
func (g *CheRouterGetter) GetSolver(client client.Client, routingClass controllerv1alpha1.WorkspaceRoutingClass) (solver solvers.RoutingSolver, err error) {
	if !isSupported(routingClass) {
		return nil, solvers.RoutingNotSupported
	}
	return &CheRoutingSolver{Client: client, scheme: g.scheme}, nil
}

// GetSpecObjects constructs cluster routing objects which should be applied on the cluster
func (c *CheRoutingSolver) GetSpecObjects(routing *controllerv1alpha1.WorkspaceRouting, workspaceMeta solvers.WorkspaceMetadata) solvers.RoutingObjects {
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

func isSupported(routingClass controllerv1alpha1.WorkspaceRoutingClass) bool {
	return routingClass == "che"
}
