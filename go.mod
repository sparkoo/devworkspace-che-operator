module github.com/che-incubator/devworkspace-che-operator

go 1.13

require (
	github.com/devfile/api v0.0.0-20201125082321-aeda60d43619
	github.com/devfile/devworkspace-operator v0.0.0-20210125082355-28b4522cab2c
	github.com/google/go-cmp v0.5.0
	github.com/openshift/api v0.0.0-20200205133042-34f0ec8dab87
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	sigs.k8s.io/controller-runtime v0.6.3
)

replace github.com/devfile/devworkspace-operator => github.com/metlos/devworkspace-operator v0.0.0-20210203210500-d41bfe6bcb51
