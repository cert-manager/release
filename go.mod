module github.com/cert-manager/release

go 1.18

require (
	cloud.google.com/go/storage v1.14.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-github/v35 v35.2.0
	github.com/google/martian v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	google.golang.org/api v0.56.0
	helm.sh/helm/v3 v3.7.0
	k8s.io/apimachinery v0.22.1
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9
	sigs.k8s.io/yaml v1.2.0
)
