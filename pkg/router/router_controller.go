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

package router

import (
	"context"
	"sync"

	"github.com/che-incubator/devworkspace-che-routing-controller/apis/che-controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/pkg/config"
	"github.com/go-logr/logr"
	routeV1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	log = ctrl.Log.WithName("router")
)

type CheRouterGetter interface {
	GetCurrentRouter(ctx context.Context) (*v1alpha1.CheRouter, error)
}

type CheRouterReconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	current *v1alpha1.CheRouter
	gateway CheGateway
	mutex   sync.Mutex
}

func (r *CheRouterReconciler) GetCurrentRouter(ctx context.Context) (*v1alpha1.CheRouter, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.current == nil {
		cur, err := r.getRouter(ctx)
		if err != nil {
			return nil, err
		}
		r.current = cur
	}

	return r.current, nil
}

func (r *CheRouterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.Log = log
	r.mutex = sync.Mutex{}

	bld := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CheRouter{}).
		Owns(&corev1.Service{}).
		Owns(&v1beta1.Ingress{})
	if config.ControllerCfg.IsOpenShift() {
		bld.Owns(&routeV1.Route{})
	}
	bld.WithEventFilter(predicate.Funcs{
		CreateFunc: func(ev event.CreateEvent) bool {
			return isCheRoute(ev.Object)
		},
		DeleteFunc: func(ev event.DeleteEvent) bool {
			return isCheRoute(ev.Object)
		},
		UpdateFunc: func(ev event.UpdateEvent) bool {
			return isCheRoute(ev.ObjectNew)
		},
		GenericFunc: func(ev event.GenericEvent) bool {
			return isCheRoute(ev.Object)
		},
	})
	return bld.Complete(r)
}

func (r *CheRouterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	// make sure we've checked we're in a valid state
	currentRouter, err := r.GetCurrentRouter(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	if currentRouter.Name != req.Name && currentRouter.Namespace != req.Namespace {
		log.Info("Ignoring reconcile request for the object because it is not the handled Che router object.", "object", req)
		return ctrl.Result{}, nil
	}

	err = r.Get(ctx, req.NamespacedName, currentRouter)
	if err != nil {
		if errors.IsNotFound(err) {
			// Ok, our current router disappeared...
			r.mutex.Lock()
			defer r.mutex.Unlock()
			r.current = nil
			return ctrl.Result{}, nil
		}
		// other error - let's requeue
		return ctrl.Result{}, err
	}

	if currentRouter.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalize(currentRouter)
	}

	if currentRouter.Spec.Routing == v1alpha1.SingleHost {
		if err = r.gateway.Sync(ctx, currentRouter); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err = r.gateway.Delete(ctx, currentRouter); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *CheRouterReconciler) getRouter(ctx context.Context) (*v1alpha1.CheRouter, error) {
	routers := v1alpha1.CheRouterList{}
	if err := r.List(ctx, &routers, nil); err != nil {
		return nil, err
	}

	var ret *v1alpha1.CheRouter = nil

	switch len(routers.Items) {
	case 0:
		log.Info("No Che Router found")
		return nil, nil
	case 1:
		ret = &routers.Items[0]
	default:
		log.Info("up to 1 CheRouter objects expected in the cluster. Using the first one", "number", len(routers.Items))
		ret = &routers.Items[0]
	}

	log.Info("Handling Che Router", "name", ret.Name, "namespace", ret.Namespace)
	return ret, nil
}

func (r *CheRouterReconciler) finalize(router *v1alpha1.CheRouter) error {
	// TODO implement
	return nil
}

func isCheRoute(obj runtime.Object) bool {
	return obj.GetObjectKind().GroupVersionKind().Kind == "CheRoute"
}
