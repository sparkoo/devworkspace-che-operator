//
// Copyright (c) 2019-2021 Red Hat, Inc.
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
	"context"
	"fmt"
	"strings"

	dwoche "github.com/che-incubator/devworkspace-che-operator/apis/che-controller/v1alpha1"
	"github.com/che-incubator/devworkspace-che-operator/pkg/defaults"
	"github.com/che-incubator/devworkspace-che-operator/pkg/sync"
	dw "github.com/devfile/api/pkg/apis/workspaces/v1alpha2"
	dwo "github.com/devfile/devworkspace-operator/apis/controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/controllers/controller/workspacerouting/solvers"
	"github.com/devfile/devworkspace-operator/pkg/common"
	"github.com/devfile/devworkspace-operator/pkg/config"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	uniqueEndpointAttributeName    = "unique"
	endpointURLPrefixPattern       = "/workspaces/%s/endpoints/%s/%d"
	uniqueEndpointURLPrefixPattern = endpointURLPrefixPattern + "-%s"
)

var (
	configMapDiffOpts = cmpopts.IgnoreFields(corev1.ConfigMap{}, "TypeMeta", "ObjectMeta")
)

func (c *CheRoutingSolver) singlehostSpecObjects(cheManager *dwoche.CheManager, routing *dwo.WorkspaceRouting, workspaceMeta solvers.WorkspaceMetadata) (solvers.RoutingObjects, error) {
	objs := solvers.RoutingObjects{}

	objs.Services = solvers.GetDiscoverableServicesForEndpoints(routing.Spec.Endpoints, workspaceMeta)

	commonService := solvers.GetServiceForEndpoints(routing.Spec.Endpoints, workspaceMeta, false, dw.PublicEndpointExposure, dw.InternalEndpointExposure)
	if commonService != nil {
		objs.Services = append(objs.Services, *commonService)
	}

	annos := map[string]string{}
	annos[defaults.ConfigAnnotationCheManagerName] = cheManager.Name
	annos[defaults.ConfigAnnotationCheManagerNamespace] = cheManager.Namespace

	additionalLabels := defaults.GetLabelsForComponent(cheManager, "exposure")

	for i := range objs.Services {
		// need to use a ref otherwise s would be a copy
		s := &objs.Services[i]

		if s.Labels == nil {
			s.Labels = map[string]string{}
		}

		for k, v := range additionalLabels {

			if len(s.Labels[k]) == 0 {
				s.Labels[k] = v
			}
		}

		if s.Annotations == nil {
			s.Annotations = map[string]string{}
		}

		for k, v := range annos {

			if len(s.Annotations[k]) == 0 {
				s.Annotations[k] = v
			}
		}
	}

	// k, now we have to create our own objects for configuring the gateway
	configMaps := c.getGatewayConfigMaps(cheManager, workspaceMeta.WorkspaceId, routing)

	syncer := sync.New(c.client, c.scheme)

	for _, cm := range configMaps {
		_, _, err := syncer.Sync(context.TODO(), nil, &cm, configMapDiffOpts)
		if err != nil {
			return solvers.RoutingObjects{}, err
		}

	}
	return objs, nil
}

func (c *CheRoutingSolver) singlehostExposedEndpoints(manager *dwoche.CheManager, workspaceID string, endpoints map[string]dwo.EndpointList, routingObj solvers.RoutingObjects) (exposedEndpoints map[string]dwo.ExposedEndpointList, ready bool, err error) {
	if manager.Status.GatewayPhase != dwoche.GatewayPhaseEstablished {
		return nil, false, nil
	}

	host := manager.Status.GatewayHost
	if host == "" {
		return nil, false, nil
	}

	exposed := map[string]dwo.ExposedEndpointList{}

	for machineName, endpoints := range endpoints {
		exposedEndpoints := dwo.ExposedEndpointList{}
		for _, endpoint := range endpoints {
			if endpoint.Exposure != dw.PublicEndpointExposure {
				continue
			}
			publicURLPrefix := getPublicURLPrefixForEndpoint(workspaceID, machineName, endpoint)
			publicURL := "https://" + ensureDoesntEndWithSlash(host) + ensureDoesntEndWithSlash(publicURLPrefix) + ensureStartsWithSlash(endpoint.Path)

			attrs := map[string]string{}
			err := endpoint.Attributes.Into(&attrs)
			if err != nil {
				return nil, false, err
			}

			exposedEndpoints = append(exposedEndpoints, dwo.ExposedEndpoint{
				Name:       endpoint.Name,
				Url:        publicURL,
				Attributes: attrs,
			})
		}
		exposed[machineName] = exposedEndpoints
	}

	return exposed, true, nil
}

func (c *CheRoutingSolver) getGatewayConfigMaps(cheManager *dwoche.CheManager, workspaceID string, routing *dwo.WorkspaceRouting) []corev1.ConfigMap {
	restrictedAnno, setRestrictedAnno := routing.Annotations[config.WorkspaceRestrictedAccessAnnotation]

	labels := defaults.GetLabelsForComponent(cheManager, "gateway-config")
	labels[config.WorkspaceIDLabel] = workspaceID
	if setRestrictedAnno {
		labels[config.WorkspaceRestrictedAccessAnnotation] = restrictedAnno
	}

	configMap := corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      defaults.GetGatewayWorkpaceConfigMapName(workspaceID),
			Namespace: cheManager.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				defaults.ConfigAnnotationWorkspaceRoutingName:      routing.Name,
				defaults.ConfigAnnotationWorkspaceRoutingNamespace: routing.Namespace,
			},
		},
		Data: map[string]string{},
	}

	routers := ""
	services := ""
	middlewares := ""

	for machineName, endpoints := range routing.Spec.Endpoints {
		// we need to support unique endpoints - so 1 port can actually be accessible
		// multiple times, each time using a different resulting external URL.
		// non-unique endpoints are all represented using a single external URL
		ports := map[int32]map[string]bool{}
		for _, e := range endpoints {
			i := int32(e.TargetPort)

			name := ""
			if e.Attributes.GetString(uniqueEndpointAttributeName, nil) == "true" {
				name = e.Name
			}

			if ports[i] == nil {
				ports[i] = map[string]bool{}
			}

			ports[i][name] = true
		}

		for port, names := range ports {
			for endpointName := range names {
				var name string
				var prefix string
				var serviceURL string

				if endpointName == "" {
					name = fmt.Sprintf("%s-%s-%d", workspaceID, machineName, port)
				} else {
					name = fmt.Sprintf("%s-%s-%d-%s", workspaceID, machineName, port, endpointName)
				}
				prefix = getPublicURLPrefix(workspaceID, machineName, port, endpointName)
				serviceURL = getServiceURL(port, workspaceID, routing.Namespace)

				// watch out for formatting - YAML must be space-idented, tabs break it. Yet go is formatted using
				// tabs.
				routers = routers + `
    ` + name + `:
      rule: PathPrefix(` + "`" + prefix + "`" + `)
      service: "` + name + `"
      middlewares:
      - "` + name + `"
      priority: 100`
				services = services + `
    ` + name + `:
      loadBalancer:
        servers:
        - url: "` + serviceURL + `"`

				middlewares = middlewares + `
    ` + name + `:
      stripPrefix:
        prefixes:
        - "` + prefix + `"`
			}
		}
	}

	configFile := `http:
  routers:` + routers + `
  services:` + services + `
  middlewares:` + middlewares + `
`

	configMap.Data[workspaceID+".yml"] = configFile

	return []corev1.ConfigMap{configMap}
}

func (c *CheRoutingSolver) singlehostFinalize(cheManager *dwoche.CheManager, routing *dwo.WorkspaceRouting) error {
	configs := &corev1.ConfigMapList{}

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", config.WorkspaceIDLabel, routing.Spec.WorkspaceId))
	if err != nil {
		return err
	}

	listOpts := &client.ListOptions{
		Namespace:     cheManager.Namespace,
		LabelSelector: selector,
	}

	err = c.client.List(context.TODO(), configs, listOpts)
	if err != nil {
		return err
	}

	for _, cm := range configs.Items {
		err = c.client.Delete(context.TODO(), &cm)
		if err != nil {
			return err
		}
	}

	return nil
}

func getServiceURL(port int32, workspaceID string, workspaceNamespace string) string {
	// the default .cluster.local suffix of the internal domain names seems to be configurable, so let's just
	// not use it so we don't have to know about it...
	return fmt.Sprintf("http://%s.%s.svc:%d", common.ServiceName(workspaceID), workspaceNamespace, port)
}

func getPublicURLPrefixForEndpoint(workspaceID string, machineName string, endpoint dw.Endpoint) string {
	endpointName := ""
	if endpoint.Attributes.GetString(uniqueEndpointAttributeName, nil) == "true" {
		endpointName = endpoint.Name
	}

	return getPublicURLPrefix(workspaceID, machineName, int32(endpoint.TargetPort), endpointName)
}

func getPublicURLPrefix(workspaceID string, machineName string, port int32, uniqueEndpointName string) string {
	if uniqueEndpointName == "" {
		return fmt.Sprintf(endpointURLPrefixPattern, workspaceID, machineName, port)
	}
	return fmt.Sprintf(uniqueEndpointURLPrefixPattern, workspaceID, machineName, port, uniqueEndpointName)
}

func ensureEndsWithSlash(str string) string {
	if strings.HasSuffix(str, "/") {
		return str
	}
	return str + "/"
}

func ensureStartsWithSlash(str string) string {
	if strings.HasPrefix(str, "/") {
		return str
	}
	return "/" + str
}

func ensureDoesntEndWithSlash(str string) string {
	return strings.TrimSuffix(str, "/")
}
