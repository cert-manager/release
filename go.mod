module github.com/cert-manager/release

go 1.13

require (
	cloud.google.com/go/storage v1.5.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-github/v29 v29.0.3
	github.com/google/martian v2.1.0+incompatible
	github.com/spf13/cobra v0.0.6
	github.com/spf13/pflag v1.0.5
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.15.0
	k8s.io/apimachinery v0.17.2
	k8s.io/release v0.2.5 // indirect
	sigs.k8s.io/yaml v1.2.0
)
