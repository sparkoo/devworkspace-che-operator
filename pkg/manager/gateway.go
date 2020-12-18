//
// Copyright (c) 2020-2020 Red Hat, Inc.
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
	"github.com/che-incubator/devworkspace-che-operator/pkg/defaults"
	"github.com/che-incubator/devworkspace-che-operator/pkg/sync"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	serviceAccountDiffOpts = cmpopts.IgnoreFields(corev1.ServiceAccount{}, "TypeMeta", "ObjectMeta", "Secrets", "ImagePullSecrets")
	roleDiffOpts           = cmpopts.IgnoreFields(rbac.Role{}, "TypeMeta", "ObjectMeta")
	roleBindingDiffOpts    = cmpopts.IgnoreFields(rbac.RoleBinding{}, "TypeMeta", "ObjectMeta")
	serviceDiffOpts        = cmp.Options{
		cmpopts.IgnoreFields(corev1.Service{}, "TypeMeta", "ObjectMeta", "Status"),
		cmpopts.IgnoreFields(corev1.ServiceSpec{}, "ClusterIP"),
	}
	configMapDiffOpts  = cmpopts.IgnoreFields(corev1.ConfigMap{}, "TypeMeta", "ObjectMeta")
	deploymentDiffOpts = cmp.Options{
		cmpopts.IgnoreFields(appsv1.Deployment{}, "TypeMeta", "ObjectMeta", "Status"),
		cmpopts.IgnoreFields(appsv1.DeploymentSpec{}, "Replicas", "RevisionHistoryLimit", "ProgressDeadlineSeconds"),
		cmpopts.IgnoreFields(appsv1.DeploymentStrategy{}, "RollingUpdate"),
		cmpopts.IgnoreFields(corev1.Container{}, "TerminationMessagePath", "TerminationMessagePolicy"),
		cmpopts.IgnoreFields(corev1.PodSpec{}, "DNSPolicy", "SchedulerName", "SecurityContext", "DeprecatedServiceAccount"),
		cmpopts.IgnoreFields(corev1.ConfigMapVolumeSource{}, "DefaultMode"),
		cmpopts.IgnoreFields(corev1.VolumeSource{}, "EmptyDir"),
		cmp.Comparer(func(x, y resource.Quantity) bool {
			return x.Cmp(y) == 0
		}),
	}
)

type CheGateway struct {
	client.Client
	Scheme *runtime.Scheme
}

func (g *CheGateway) Sync(ctx context.Context, router *v1alpha1.CheManager) error {

	syncer := sync.Syncer{Client: g.Client, Scheme: g.Scheme}

	sa := getGatewayServiceAccountSpec(router)
	if _, err := syncer.Sync(ctx, router, &sa, serviceAccountDiffOpts); err != nil {
		return err
	}

	role := getGatewayRoleSpec(router)
	if _, err := syncer.Sync(ctx, router, &role, roleDiffOpts); err != nil {
		return err
	}

	roleBinding := getGatewayRoleBindingSpec(router)
	if _, err := syncer.Sync(ctx, router, &roleBinding, roleBindingDiffOpts); err != nil {
		return err
	}

	traefikConfig := getGatewayTraefikConfigSpec(router)
	if _, err := syncer.Sync(ctx, router, &traefikConfig, configMapDiffOpts); err != nil {
		return err
	}

	depl := getGatewayDeploymentSpec(router)
	if _, err := syncer.Sync(ctx, router, &depl, deploymentDiffOpts); err != nil {
		return err
	}

	service := getGatewayServiceSpec(router)
	if _, err := syncer.Sync(ctx, router, &service, serviceDiffOpts); err != nil {
		return err
	}

	return nil
}

func (g *CheGateway) Delete(ctx context.Context, router *v1alpha1.CheManager) error {
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
		},
	}
	if err := g.delete(ctx, &deployment); err != nil {
		return err
	}

	serverConfig := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
		},
	}
	if err := g.delete(ctx, &serverConfig); err != nil {
		return err
	}

	roleBinding := rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
		},
	}
	if err := g.delete(ctx, &roleBinding); err == nil {
		return err
	}

	role := rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
		},
	}
	if err := g.delete(ctx, &role); err == nil {
		return err
	}

	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
		},
	}
	if err := g.delete(ctx, &sa); err == nil {
		return err
	}

	return nil
}

func (g *CheGateway) delete(ctx context.Context, obj metav1.Object) error {
	key := client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	ro := obj.(runtime.Object)
	if getErr := g.Get(ctx, key, ro); getErr == nil {
		if err := g.Client.Delete(ctx, ro); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

// below functions declare the desired states of the various objects required for the gateway

func getGatewayServiceAccountSpec(router *v1alpha1.CheManager) corev1.ServiceAccount {
	return corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
			Labels:    defaults.GetLabels(router, "security"),
		},
	}
}

func getGatewayRoleSpec(router *v1alpha1.CheManager) rbac.Role {
	return rbac.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbac.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
			Labels:    defaults.GetLabels(router, "security"),
		},
		Rules: []rbac.PolicyRule{
			{
				Verbs:     []string{"watch", "get", "list"},
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
			},
		},
	}
}

func getGatewayRoleBindingSpec(router *v1alpha1.CheManager) rbac.RoleBinding {
	return rbac.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbac.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
			Labels:    defaults.GetLabels(router, "security"),
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     router.Name,
		},
		Subjects: []rbac.Subject{
			{
				Kind: "ServiceAccount",
				Name: router.Name,
			},
		},
	}
}

func getGatewayTraefikConfigSpec(router *v1alpha1.CheManager) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
			Labels:    defaults.GetLabels(router, "gateway-config"),
		},
		Data: map[string]string{
			"traefik.yml": `
entrypoints:
  http:
    address: ":8080"
    forwardedHeaders:
      insecure: true
  https:
    address: ":8443"
    forwardedHeaders:
      insecure: true
global:
  checkNewVersion: false
  sendAnonymousUsage: false
providers:
  file:
    directory: "/dynamic-config"
    watch: true
log:
  level: "INFO"`,
		},
	}
}

func getGatewayDeploymentSpec(router *v1alpha1.CheManager) appsv1.Deployment {
	gatewayImage := defaults.GetGatewayImage()
	sidecarImage := defaults.GetGatewayConfigurerImage()

	terminationGracePeriodSeconds := int64(10)

	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
			Labels:    defaults.GetLabels(router, "deployment"),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: defaults.GetLabels(router, "deployment"),
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaults.GetLabels(router, "deployment"),
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					ServiceAccountName:            router.Name,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:            "gateway",
							Image:           gatewayImage,
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "static-config",
									MountPath: "/etc/traefik",
								},
								{
									Name:      "dynamic-config",
									MountPath: "/dynamic-config",
								},
							},
						},
						{
							Name:            "configbump",
							Image:           sidecarImage,
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dynamic-config",
									MountPath: "/dynamic-config",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "CONFIG_BUMP_DIR",
									Value: "/dynamic-config",
								},
								{
									Name:  "CONFIG_BUMP_LABELS",
									Value: labels.FormatLabels(defaults.GetLabels(router, "gateway-config")),
								},
								{
									Name: "CONFIG_BUMP_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.namespace",
										},
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "static-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: router.Name,
									},
								},
							},
						},
						{
							Name: "dynamic-config",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}

func getGatewayServiceSpec(router *v1alpha1.CheManager) corev1.Service {
	return corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      router.Name,
			Namespace: router.Namespace,
			Labels:    defaults.GetLabels(router, "deployment"),
		},
		Spec: corev1.ServiceSpec{
			Selector:        defaults.GetLabels(router, "deployment"),
			SessionAffinity: corev1.ServiceAffinityNone,
			Type:            corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "gateway-http",
					Port:       8080,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8080),
				},
				{
					Name:       "gateway-https",
					Port:       8443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8443),
				},
			},
		},
	}
}
