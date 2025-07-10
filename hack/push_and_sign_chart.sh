# Copyright 2021 The cert-manager Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/bash

set -eu -o pipefail

# This script downloads the cert-manager Helm chart for a given release version (provided via the RELEASE_VERSION environment variable),
# and then pushes the chart to the cert-manager OCI registry at quay.io/jetstack/charts along with its .prov signature file.
# Finally, it signs the tag using cosign.
#
# This script assumes that you're already authenticated with the Google Cloud Platform (GCP) project cert-manager-release for downloading
# the chart and for using the KMS key for signing.
# It also assumes that you have the gsutil, helm, and cosign commands installed and available in your PATH.

# WARNING: Release assets for versions earlier than v1.8.1 have a different path structure and you'll need to manually set the release version
# to the correct value. On the assumption that this script will almost always used for v1.8.1 and later releases, we don't check this.
# If you need to release a really old version, you can find the path structure in the cert-manager-release GCS bucket manually.
# Releases before v1.6.0 do not have a .prov file, so you should set the SKIP_PROV environment variable to "true" when running this script in that case.
#
# This script is idempotent and should be safe to run multiple times for the same RELEASE_VERSION; multiple invocations
# may create multiple cosign signatures, but the underlying operation of pushing the chart will not fail if the chart already
# exists in the registry.
#
# See the important note near the bottom about the cosign "--tlog-upload=false" flag.

if [ -z "$RELEASE_VERSION" ]; then
  echo "Error: RELEASE_VERSION environment variable is not set."
  exit 1
fi

if ! command -v gsutil >/dev/null 2>&1; then
  echo "Error: gsutil is not installed."
  exit 1
fi

if ! command -v cosign >/dev/null 2>&1; then
  echo "Error: cosign is not installed."
  exit 1
fi

TEMP_DIR=$(mktemp -d "/tmp/cert-manager-release-${RELEASE_VERSION}-XXXXXX")

echo "Temporary directory created: $TEMP_DIR"

cleanup() {
  echo "Cleaning up temporary directory: $TEMP_DIR"
  rm -rf "$TEMP_DIR"
}

trap cleanup EXIT

echo "Downloading release assets"
gsutil cp "gs://cert-manager-release/stage/gcb/release/$RELEASE_VERSION/cert-manager-manifests.tar.gz" $TEMP_DIR/cert-manager-manifests.tar.gz

echo "Extracting cert-manager Helm chart"
tar xfO $TEMP_DIR/cert-manager-manifests.tar.gz deploy/chart/cert-manager-$RELEASE_VERSION.tgz > $TEMP_DIR/cert-manager-$RELEASE_VERSION.tgz

if [ "${SKIP_PROV:-false}" != "true" ]; then
  echo "Extracting Helm chart signature"
  tar xfO $TEMP_DIR/cert-manager-manifests.tar.gz deploy/chart/cert-manager-$RELEASE_VERSION.tgz.prov > $TEMP_DIR/cert-manager-$RELEASE_VERSION.tgz.prov
else
  echo "Skipping extraction of Helm chart signature as SKIP_PROV is set to true; this should only be done for cert-manager releases that do not have a .prov file, i.e. earlier than v1.6.0"
fi

echo "Pushing cert-manager chart to OCI registry"

# NB: helm push also pushes the corresponding .prov file, so we don't need to do that explicitly.
# See https://helm.sh/docs/topics/registries/#the-push-subcommand for details on .prov pushing
# The .prov file is referenced transparently in the chart's OCI manifest; you won't see a tag for it.
# To verify that the .prov file was pushed, you can run:
# crane manifest quay.io/jetstack/charts/cert-manager:$RELEASE_VERSION | jq
# and look for the "application/vnd.cncf.helm.chart.provenance.v1.prov" entry in the layers.
helm push $TEMP_DIR/cert-manager-$RELEASE_VERSION.tgz oci://quay.io/jetstack/charts

# The key is taken from cmd/cmrel/cmd/const.go
# You may need to update it here if the key changes in that file.

COSIGN_KEY="gcpkms://projects/cert-manager-release/locations/europe-west1/keyRings/cert-manager-release/cryptoKeys/cert-manager-release-signing-key/cryptoKeyVersions/1"

# Why do we use --tlog-upload=false?
# This flag prevents us creating a tlog entry for the signature, which is usually a good thing to do.
# Unfortunately, as well as creating the tlog entry, cosign also attempts to verify the tlog entry,
# which is the issue we run into - our KMS key uses SHA-512 as the signature digest algorithm, but there's no option
# to specify the digest algorithm for the tlog entry, so verification fails.
# We solved this for "cosign verify" with a cosign PR[0] a while back, but this problem hasn't been solved for tlog verification.
# [0]: https://github.com/sigstore/cosign/pull/1071

echo "Signing chart with cosign"
cosign sign --key $COSIGN_KEY \
  --tlog-upload=false \
  quay.io/jetstack/charts/cert-manager:$RELEASE_VERSION

# See the above comment for why we use --insecure-ignore-tlog=true
# We do cosign verify here as a sanity check.
cosign verify --key $COSIGN_KEY --signature-digest-algorithm sha512 --insecure-ignore-tlog=true quay.io/jetstack/charts/cert-manager:$RELEASE_VERSION
