module github.com/che-incubator/devworkspace-che-operator

go 1.13

require (
	github.com/devfile/devworkspace-operator v0.0.0
	github.com/go-logr/logr v0.1.0
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/google/go-cmp v0.5.0
	github.com/openshift/api v0.0.0-20200205133042-34f0ec8dab87
	github.com/prometheus/client_golang v1.3.0 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	sigs.k8s.io/controller-runtime v0.6.3
)

replace github.com/devfile/devworkspace-operator => github.com/amisevsk/devworkspace-operator v0.0.0-20201211032043-0c70133bee40
