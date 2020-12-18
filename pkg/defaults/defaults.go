package defaults

import (
	"os"
	"runtime"

	"github.com/che-incubator/devworkspace-che-operator/apis/che-controller/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	gatewayImageEnvVarName           = "RELATED_IMAGE_gateway"
	gatewayConfigurerImageEnvVarName = "RELATED_IMAGE_gateway_configurer"

	defaultGatewayImage           = "docker.io/traefik:v2.2.8"
	defaultGatewayConfigurerImage = "quay.io/che-incubator/configbump:0.1.4"
)

var (
	log = ctrl.Log.WithName("defaults")
)

func GetLabels(router *v1alpha1.Che, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      router.Name,
		"app.kubernetes.io/part-of":   router.Name,
		"app.kubernetes.io/component": component,
	}
}

func GetGatewayImage() string {
	return read(gatewayImageEnvVarName, defaultGatewayImage)
}

func GetGatewayConfigurerImage() string {
	return read(gatewayConfigurerImageEnvVarName, defaultGatewayConfigurerImage)
}

func read(varName string, fallback string) string {
	ret := os.Getenv(varName)

	if len(ret) == 0 {
		ret = os.Getenv(archDependent(varName))
		if len(ret) == 0 {
			log.Info("Failed to read the default value from the environment. Will use the hardcoded default value.", "envvar", varName, "value", fallback)
			ret = fallback
		}
	}

	return ret
}

func archDependent(envVarName string) string {
	return envVarName + "_" + runtime.GOARCH
}
