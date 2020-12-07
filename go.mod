module github.com/ShotaKitazawa/pipecd-operator

go 1.13

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/jessevdk/go-assets v0.0.0-20160921144138-4f4301a06e15
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pipe-cd/pipe v0.9.0
	google.golang.org/grpc v1.31.1
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
	sigs.k8s.io/controller-runtime v0.5.0
)

replace (
	k8s.io/api => k8s.io/api v0.17.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.2
	k8s.io/client-go => k8s.io/client-go v0.17.2
)
