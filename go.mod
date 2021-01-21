module github.com/symcn/meshach

go 1.13

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/DeanThompson/ginpprof v0.0.0-20190408063150-3be636683586
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/gin-gonic/gin v1.6.3
	github.com/go-logr/logr v0.1.0
	github.com/go-zookeeper/zk v1.0.2
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.3.1
	github.com/golangci/golangci-lint v1.35.2 // indirect
	github.com/nacos-group/nacos-sdk-go v0.3.2
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	go.opencensus.io v0.22.2
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/tools v0.0.0-20210115202250-e0d201561e39 // indirect
	google.golang.org/grpc v1.28.1
	istio.io/api v0.0.0-20200521171657-32375f234cc1
	istio.io/client-go v0.0.0-20200518164621-ef682e2929e5
	istio.io/istio v0.0.0-20200525040124-4ed2c700a1d9
	k8s.io/api v0.18.3
	k8s.io/apiextensions-apiserver v0.18.0
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/helm v2.16.3+incompatible
	k8s.io/klog v1.0.0
	k8s.io/sample-controller v0.17.4
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/api => k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.4
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
