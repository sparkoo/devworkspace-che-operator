module github.com/che-incubator/devworkspace-che-routing-controller

go 1.13

require (
	github.com/devfile/devworkspace-operator v0.0.0
	github.com/go-kit/kit v0.10.0
	github.com/go-logr/logr v0.1.0
	go.uber.org/zap v1.16.0 // indirect
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	sigs.k8s.io/controller-runtime v0.6.3
)

replace github.com/devfile/devworkspace-operator => github.com/amisevsk/devworkspace-operator v0.0.0-20201210050715-ffc7d2f087ed
