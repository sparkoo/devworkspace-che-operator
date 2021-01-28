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
	"sync"

	"github.com/che-incubator/devworkspace-che-operator/apis/che-controller/v1alpha1"
	"github.com/che-incubator/devworkspace-che-operator/pkg/gateway"
	"github.com/che-incubator/devworkspace-che-operator/pkg/infrastructure"
	datasync "github.com/che-incubator/devworkspace-che-operator/pkg/sync"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log             = ctrl.Log.WithName("che")
	currentManagers = map[client.ObjectKey]v1alpha1.CheManager{}
	managerAccess   = sync.Mutex{}
)

type CheReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	gateway gateway.CheGateway
	syncer  datasync.Syncer
}

func GetCurrentManagers() map[client.ObjectKey]v1alpha1.CheManager {
	managerAccess.Lock()
	defer managerAccess.Unlock()

	ret := map[client.ObjectKey]v1alpha1.CheManager{}

	for k, v := range currentManagers {
		ret[k] = v
	}

	return ret
}

func (r *CheReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.client = mgr.GetClient()
	r.scheme = mgr.GetScheme()
	r.gateway = gateway.New(mgr.GetClient(), mgr.GetScheme())
	r.syncer = datasync.New(r.client, r.scheme)

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
	if infrastructure.Current.Type == infrastructure.OpenShift {
		bld.Owns(&routev1.Route{})
	}
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

	var changed bool
	var host string

	if changed, host, err = r.reconcileGateway(ctx, current); err != nil {
		return ctrl.Result{}, err
	}

	res, err := r.updateStatus(ctx, current, changed, host)

	if err == nil {
		// update the shared map
		managerAccess.Lock()
		defer managerAccess.Unlock()

		currentManagers[req.NamespacedName] = *current
	}

	return res, err
}

func (r *CheReconciler) updateStatus(ctx context.Context, manager *v1alpha1.CheManager, changed bool, host string) (ctrl.Result, error) {
	currentPhase := manager.Status.GatewayPhase
	currentHost := manager.Status.GatewayHost

	if manager.Spec.Routing == v1alpha1.MultiHost {
		manager.Status.GatewayPhase = v1alpha1.GatewayPhaseInactive
	} else if changed {
		manager.Status.GatewayPhase = v1alpha1.GatewayPhaseInitializing
	} else {
		manager.Status.GatewayPhase = v1alpha1.GatewayPhaseEstablished
	}

	manager.Status.GatewayHost = host

	if currentPhase != manager.Status.GatewayPhase || currentHost != manager.Status.GatewayHost {
		return ctrl.Result{Requeue: true}, r.client.Status().Update(ctx, manager)
	}

	return ctrl.Result{Requeue: currentPhase == v1alpha1.GatewayPhaseInitializing}, nil
}

func (r *CheReconciler) finalize(router *v1alpha1.CheManager) error {
	// implement if needed
	return nil
}

func (r *CheReconciler) reconcileGateway(ctx context.Context, manager *v1alpha1.CheManager) (bool, string, error) {
	var changed bool
	var err error
	var host string

	if manager.Spec.Routing == v1alpha1.SingleHost {
		changed, host, err = r.gateway.Sync(ctx, manager)
	} else {
		changed, host, err = true, "", r.gateway.Delete(ctx, manager)
	}

	return changed, host, err
}
