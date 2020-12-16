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

package main

import (
	"flag"
	"os"

	controllerv1alpha1 "github.com/devfile/devworkspace-operator/apis/controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/controllers/controller/workspacerouting"
	"github.com/devfile/devworkspace-operator/controllers/controller/workspacerouting/solvers"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/che-incubator/devworkspace-che-routing-controller/pkg/router"
	"github.com/che-incubator/devworkspace-che-routing-controller/pkg/solver"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var _ router.CheRouterGetter = (*router.CheRouterReconciler)(nil)

func init() {
	controllerv1alpha1.AddToScheme(scheme)
	extensions.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
}

func main() {

	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "8d217f94.devfile.io",
	})

	if err != nil {
		setupLog.Error(err, "unable to start the operator manager")
		os.Exit(1)
	}

	routerReconciler := &router.CheRouterReconciler{}
	if err = routerReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CheRouter")
		os.Exit(1)
	}

	routingReconciler := &workspacerouting.WorkspaceRoutingReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("CheWorkspaceRouting"),
		Scheme: mgr.GetScheme(),
		GetSolverFunc: func(routingClass controllerv1alpha1.WorkspaceRoutingClass) (solvers.RoutingSolver, error) {
			if routingClass != "che" {
				return nil, workspacerouting.RoutingNotSupported
			}

			var getter router.CheRouterGetter = routerReconciler

			return solver.New(mgr.GetClient(), mgr.GetScheme(), &getter), nil
		},
	}
	if err = routingReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CheWorkspaceRoutingSolver")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
