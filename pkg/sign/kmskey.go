/*
Copyright 2021 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sign

import (
	"fmt"
	"regexp"
)

var keyRegex = regexp.MustCompile(`^projects/([^/]+)/locations/([^/]+)/keyRings/([^/]+)/cryptoKeys/([^/]+)/cryptoKeyVersions/([^/]+)$`)

// GCPKMSKey holds a GCP KMS key, easily serializable to either GCP format ('cryptoKeyVersions') or cosign format ('versions')
type GCPKMSKey struct {
	projectID  string
	locationID string
	keyRing    string
	keyName    string
	version    string
}

// NewGCPKMSKey parses and validates an input KMS key. The accepted format is that provided when copying the resource name in the GCP console.
// The format provided by GCP is distinct from the format required by cosign; notably
// GCP uses "cryptoKeyVersions" and cosign requires "versions".
func NewGCPKMSKey(raw string) (GCPKMSKey, error) {
	v := keyRegex.FindStringSubmatch(raw)

	if len(v) != 6 {
		return GCPKMSKey{}, fmt.Errorf("invalid GCP KMS format: %q", raw)
	}

	projectID, locationID, keyRing, keyName, version := v[1], v[2], v[3], v[4], v[5]

	return GCPKMSKey{
		projectID:  projectID,
		locationID: locationID,
		keyRing:    keyRing,
		keyName:    keyName,
		version:    version,
	}, nil
}

// String returns the key in GCP format
func (g GCPKMSKey) String() string {
	return g.GCPFormat()
}

// GCPFormat returns the key verbatim, which will be the format required for GCP actions
func (g GCPKMSKey) GCPFormat() string {
	return fmt.Sprintf(
		"projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/%s",
		g.projectID,
		g.locationID,
		g.keyRing,
		g.keyName,
		g.version,
	)
}

// CosignFormat returns the key in the correct format for cosign, which uses "versions" instead of "cryptoKeyVersions". Also prepends the gcpkms scheme
func (g GCPKMSKey) CosignFormat() string {
	return fmt.Sprintf(
		"gcpkms://projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/versions/%s",
		g.projectID,
		g.locationID,
		g.keyRing,
		g.keyName,
		g.version,
	)
}
