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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoutingType string

const (
	SingleHost RoutingType = "singlehost"
	MultiHost  RoutingType = "multihost"
)

// CheRouterSpec holds the configuration of the Che routing controller.
// Currently on a single object of this type is handled in the cluster.
// +k8s:openapi-gen=true
type CheRouterSpec struct {
	// The hostname to use for creating the workspace endpoints
	// This is used as a full hostname in the singlehost mode. In the multihost mode, the individual
	// endpoints are exposed on subdomains of the specified host.
	Host string `json:"host,omitempty"`

	// Routing defines how the Che Router exposes the workspaces and components within
	Routing RoutingType `json:"routing,omitempty"`

	// GatewayImage is the docker image to use for the Che gateway. If not defined in the CR,
	// it is taken from the `RELATED_IMAGE_gateway` environment variable of the che routing
	// operator deployment/pod. If not defined there it defaults to a hardcoded value.
	GatewayImage string `json:"gatewayImage,omitempty"`

	// GatewayConfigureImage is the docker image to use for the sidecar of the Che gateway that is
	// used to configure it. If not defined in the CR, it is taken from the `RELATED_IMAGE_gateway_configurer`
	// environment variable of the che routing operator deployment/pod. If not defined there it defaults to
	// a hardcoded value.
	GatewayConfigurerImage string `json:"gatewayConfigurerImage,omitempty"`

	// TODO in the future we could theoretically add support for having multiple che routers
	// by being able to specify an explicit routing class here for each router. Alternatively,
	// we could just use the router's name as the routing class for example.
}

// +k8s:openapi-gen=true
type CheRouterStatus struct {
	GatewayPhase string `json:"gatewayPhase,omitempty"`
}

// CheRouter is the schema for the che router api
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cherouters,scope=Namespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CheRouter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CheRouterSpec   `json:"spec,omitempty"`
	Status CheRouterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CheRouterList is the list type for CheRouter
type CheRouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CheRouter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CheRouter{}, &CheRouterList{})
}
