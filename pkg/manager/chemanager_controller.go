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

package manager

import (
	"context"

	"github.com/che-incubator/devworkspace-che-operator/apis/che-controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/pkg/config"
	routeV1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	log = ctrl.Log.WithName("che")
)

type CheReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	gateway CheGateway
}

func (r *CheReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.client = mgr.GetClient()
	r.scheme = mgr.GetScheme()
	r.gateway.client = mgr.GetClient()
	r.gateway.scheme = mgr.GetScheme()

	bld := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CheManager{}).
		Owns(&corev1.Service{}).
		Owns(&v1beta1.Ingress{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbac.Role{}).
		Owns(&rbac.RoleBinding{})
	if config.ControllerCfg.IsOpenShift() {
		bld.Owns(&routeV1.Route{})
	}
	bld.WithEventFilter(predicate.Funcs{
		CreateFunc: func(ev event.CreateEvent) bool {
			return isChe(ev.Object)
		},
		DeleteFunc: func(ev event.DeleteEvent) bool {
			return isChe(ev.Object)
		},
		UpdateFunc: func(ev event.UpdateEvent) bool {
			return isChe(ev.ObjectNew)
		},
		GenericFunc: func(ev event.GenericEvent) bool {
			return isChe(ev.Object)
		},
	})
	return bld.Complete(r)
}

func (r *CheReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	// make sure we've checked we're in a valid state
	current := &v1alpha1.CheManager{}
	err := r.client.Get(ctx, req.NamespacedName, current)
	if err != nil {
		if errors.IsNotFound(err) {
			// Ok, our current router disappeared...
			return ctrl.Result{}, nil
		}
		// other error - let's requeue
		return ctrl.Result{}, err
	}

	if current.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalize(current)
	}

	if current.Spec.Routing == v1alpha1.SingleHost {
		if err = r.gateway.Sync(ctx, current); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err = r.gateway.Delete(ctx, current); err != nil {
			return ctrl.Result{}, err
		}
	}

	// TODO create ingress/route according to current.Host

	return ctrl.Result{}, nil
}

func (r *CheReconciler) finalize(router *v1alpha1.CheManager) error {
	// implement if needed
	return nil
}

func isChe(obj runtime.Object) bool {
	return obj.GetObjectKind().GroupVersionKind().Kind == "Che"
}
