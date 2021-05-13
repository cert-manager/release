module github.com/cert-manager/release

go 1.16

require (
	cloud.google.com/go/storage v1.14.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-github/v35 v35.2.0
	github.com/google/martian v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84
	google.golang.org/api v0.43.0
	k8s.io/apimachinery v0.20.5
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10
	sigs.k8s.io/yaml v1.2.0
)
