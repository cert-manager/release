package validation

import (
	"reflect"
	"testing"

	"github.com/cert-manager/release/pkg/release"
	"github.com/cert-manager/release/pkg/release/images"
	"github.com/cert-manager/release/pkg/release/images/fake"
)

func TestValidate_Semver(t *testing.T) {
	for _, test := range []struct {
		version    string
		violations []string
		err        string
	}{
		{
			version:    "v0.15",
			violations: []string{`Release version "v0.15" is not semver compliant: No Major.Minor.Patch elements found`},
		},
		{
			version:    "v0.15-beta.0",
			violations: []string{`Release version "v0.15-beta.0" is not semver compliant: Invalid character(s) found in minor number "15-beta"`},
		},
		{
			version: "v0.15.0",
		},
		{
			version: "v0.15.0-beta.0",
		},
		{
			version: "v0.15.0-beta.0-2",
		},
		{
			version:    "0.15.0-beta.0-2",
			violations: []string{`Release version "0.15.0-beta.0-2" is not semver compliant: version number must have a leading 'v' character`},
		},
	} {
		t.Run("version_"+test.version, func(t *testing.T) {
			v, err := ValidateUnpackedRelease(Options{}, &release.Unpacked{
				ReleaseVersion: test.version,
			})
			if err == nil && test.err != "" {
				t.Errorf("error did not match expected: got=%v, exp=%v", err, test.err)
			}
			if err != nil && err.Error() != test.err {
				t.Errorf("error did not match expected: got=%v, exp=%v", err, test.err)
			}
			if !reflect.DeepEqual(v, test.violations) {
				t.Errorf("unexpected violations: got=%v, exp=%v", v, test.violations)
			}
		})
	}
}

func Test_validateImageBundles(t *testing.T) {
	type args struct {
		bundles map[string][]images.TarInterface
		opts    Options
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "no errors on a correct image name",
			args: args{
				bundles: map[string][]images.TarInterface{"controller": []images.TarInterface{fake.New("quay.io/jetstack/cert-manager-controller-amd64", "v0.15.0", "dummy", "linux", "amd64", "amd64")}},
				opts: Options{
					ReleaseVersion:  "v0.15.0",
					ImageRepository: "quay.io/jetstack",
				},
			},
			want: nil,
		},
		{
			name: "error on incorrect image name",
			args: args{
				bundles: map[string][]images.TarInterface{"controller": []images.TarInterface{fake.New("nginx", "v0.15.0", "dummy", "linux", "amd64", "amd64")}},
				opts: Options{
					ReleaseVersion:  "v0.15.0",
					ImageRepository: "quay.io/jetstack",
				},
			},
			want: []string{`Image "nginx" does not match expected name "quay.io/jetstack/cert-manager-controller-amd64"`},
		},
		{
			name: "error on incorrect image tag",
			args: args{
				bundles: map[string][]images.TarInterface{"controller": []images.TarInterface{fake.New("quay.io/jetstack/cert-manager-controller-amd64", "v0.8.0", "dummy", "linux", "amd64", "amd64")}},
				opts: Options{
					ReleaseVersion:  "v0.15.0",
					ImageRepository: "quay.io/jetstack",
				},
			},
			want: []string{`Image "quay.io/jetstack/cert-manager-controller-amd64" does not have expected tag "v0.15.0"`},
		},
		{
			name: "error on incorrect image architecture",
			args: args{
				bundles: map[string][]images.TarInterface{"controller": []images.TarInterface{fake.New("quay.io/jetstack/cert-manager-controller-arm", "v0.15.0", "dummy", "linux", "arm", "amd64")}},
				opts: Options{
					ReleaseVersion:  "v0.15.0",
					ImageRepository: "quay.io/jetstack",
				},
			},
			want: []string{`Image architecture "amd64" does not match expected architecture "arm"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateImageBundles(tt.args.bundles, tt.args.opts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateImageBundles() = %v, want %v", got, tt.want)
			}
		})
	}
}
