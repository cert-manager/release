#!/usr/bin/env bash
# Copyright 2020 The Jetstack cert-manager contributors.
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

echo "+++ Building cmrel tool"
go build -o cmrel ./cmd/cmrel

# clone cert-manager @ master
echo "+++ Cloning jetstack/cert-manager repository"
tmpdir="$(mktemp -d)"
trap "rm -rf ${tmpdir}" EXIT
git clone https://github.com/jetstack/cert-manager.git "${tmpdir}"

echo "+++ Running 'gcb stage' command"
./cmrel gcb stage \
  --repo-path="${tmpdir}" \
  --skip-push=true \
  --debug
