module github.com/submariner-io/coastguard

go 1.12

require (
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/submariner-io/submariner v0.0.0-20190708095718-350482d85dd4
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/client-go v0.0.0-20190521190702-177766529176
	k8s.io/klog v0.0.0-20181108234604-8139d8cb77af
)

replace (
	github.com/bronze1man/goStrongswanVici => github.com/mangelajo/goStrongswanVici v0.0.0-20190223031456-9a5ae4453bd
	github.com/submariner-io/submariner => github.com/tpantelis/submariner v0.0.2-0.20190715192543-9fb42e3d1fb3
)
