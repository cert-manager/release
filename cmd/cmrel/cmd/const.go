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

package cmd

// defaultKMSKey is the default signing key; this shouldn't change often so it should be safe enough
// to hardcode it as a default for the quality-of-life improvement it brings to invoking various cmrel commands
// WARNING: cosign requires a different format for the key; this is the format required by the GCP API but not cosign (which needs "versions" instead of "cryptoKeyVersions")
const defaultKMSKey = "projects/cert-manager-release/locations/europe-west1/keyRings/cert-manager-release/cryptoKeys/cert-manager-release-signing-key/cryptoKeyVersions/1"
