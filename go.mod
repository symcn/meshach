module github.com/mesh-operator

go 1.13

require (
	github.com/nacos-group/nacos-sdk-go v0.3.2
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/pkg/errors v0.9.1
	github.com/samuel/go-zookeeper v0.0.0-20190923202752-2cc03de413da
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	google.golang.org/grpc v1.29.1
	istio.io/api v0.0.0-20200512234804-e5412c253ffe
	istio.io/istio v0.0.0-20200514064816-577b897f89dc
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
