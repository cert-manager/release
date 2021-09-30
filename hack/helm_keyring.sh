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

#!/usr/bin/env bash

set -eu -o pipefail

# This script imports $1 into gpg then exports a helm-compatible keyring at $2

KEY_FILE=${1:-}
KEYRING_FILE=${2:-}

if [ x"$KEY_FILE" == "x" ] ||  [ x"$KEYRING_FILE" == "x" ]; then
	echo "missing argument: $0 <key file> <keyring output file>"
	exit 1
fi

if [ ! -f "$KEY_FILE" ]; then
	echo "'$KEY_FILE' doesn't seem to exist, exiting"
	exit 1
fi

FINGERPRINT=$(gpg --with-colons --with-fingerprint < $KEY_FILE 2>/dev/null | grep fpr | cut -d ":" -f10)

echo "key has fingerprint: $FINGERPRINT"

gpg --batch --import < $KEY_FILE

gpg --export "$FINGERPRINT" > $KEYRING_FILE

echo "wrote '$KEYRING_FILE'"
