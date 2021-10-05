#!/usr/bin/env bash

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

set -o nounset
set -o errexit
set -o pipefail

CMREL=${1:-}

SIGNING_KEY=${2:-}
SKIP_SIGNING="true"

if [ -z $CMREL ]; then
	echo "usage: $0 <path-to-cmrel>"
	exit 1
fi

if [ ! -z $SIGNING_KEY ]; then
	SKIP_SIGNING="false"
fi

# clone cert-manager @ master
echo "+++ Cloning jetstack/cert-manager repository"
tmpdir="$(mktemp -d)"
trap "rm -rf ${tmpdir}" EXIT
git clone https://github.com/jetstack/cert-manager.git "${tmpdir}"

echo "+++ Running 'gcb stage' command"
$CMREL gcb stage \
  --repo-path="${tmpdir}" \
  --skip-push=true \
  --signing-kms-key="${SIGNING_KEY}" \
  --skip-signing="${SKIP_SIGNING}" \
  --debug
