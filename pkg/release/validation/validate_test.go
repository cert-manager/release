package validation

import (
	"reflect"
	"testing"

	"github.com/cert-manager/release/pkg/release"
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
