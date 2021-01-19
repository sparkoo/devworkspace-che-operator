package manager

import (
	"context"

	"github.com/che-incubator/devworkspace-che-operator/apis/che-controller/v1alpha1"
	"github.com/che-incubator/devworkspace-che-operator/pkg/defaults"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	generatedHostAnnotation = "openshift.io/host.generated"
)

var (
	// used when the che manager spec defines the host explicitly
	explicitHostRouteDiffOpts = cmp.Options{
		cmpopts.IgnoreFields(routev1.Route{}, "TypeMeta", "ObjectMeta", "Status"),
		cmpopts.IgnoreFields(routev1.RouteSpec{}, "WildcardPolicy"),
		cmpopts.IgnoreFields(routev1.RouteTargetReference{}, "Weight"),
	}

	generatedHostRouteDiffOpts = cmp.Options{
		cmpopts.IgnoreFields(routev1.Route{}, "TypeMeta", "ObjectMeta", "Status"),
		cmpopts.IgnoreFields(routev1.RouteSpec{}, "WildcardPolicy", "Host"),
		cmpopts.IgnoreFields(routev1.RouteTargetReference{}, "Weight"),
	}
)

func (r *CheReconciler) reconcileRoute(ctx context.Context, manager *v1alpha1.CheManager) (bool, error) {
	route := getRouteSpec(manager)
	var changed bool
	var err error

	if manager.Spec.Routing != v1alpha1.SingleHost {
		changed, err = true, r.syncer.Delete(ctx, route)
	} else {
		// first try to get the route and see if we have the "explicit-host" anno set on it.
		// existing = generated, now = generated -> sync without host
		// existing = generated, now = explicit -> re-create the route
		// existing = explicit, now = generated -> re-create the route
		// existing = explicit, now = explicit -> sync with host

		expectedGeneratedHost := manager.Spec.Host == ""

		key := client.ObjectKey{Name: route.Name, Namespace: route.Namespace}
		existing := &routev1.Route{}
		if err := r.client.Get(ctx, key, existing); err != nil {
			if !errors.IsNotFound(err) {
				return false, err
			}
		}

		existingGeneratedHostValue := existing.Annotations[generatedHostAnnotation]
		var existingGeneratedHost bool

		if existingGeneratedHostValue == "" || existingGeneratedHostValue == "false" {
			existingGeneratedHost = false
		} else {
			existingGeneratedHost = true
		}

		if existingGeneratedHost != expectedGeneratedHost {
			r.syncer.Delete(ctx, route)
		}

		var diffOpts cmp.Options
		if manager.Spec.Host == "" {
			diffOpts = generatedHostRouteDiffOpts
		} else {
			diffOpts = explicitHostRouteDiffOpts
		}

		changed, _, err = r.syncer.Sync(ctx, manager, route, diffOpts)
	}

	return changed, err
}

func getRouteSpec(manager *v1alpha1.CheManager) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manager.Name,
			Namespace: manager.Namespace,
			Labels:    defaults.GetLabelsForComponent(manager, "external-access"),
		},
		Spec: routev1.RouteSpec{
			Host: manager.Spec.Host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: GetGatewayServiceName(manager),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(GatewayPort),
			},
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   routev1.TLSTerminationEdge,
			},
		},
	}
}
