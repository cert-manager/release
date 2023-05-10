#!/usr/bin/env bash

# Copyright 2023 The cert-manager Authors.
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

set -eu -o pipefail

# This script tests the validate-gomod subcommand for cmrel against a local, intentionally-broken repo layout

CMREL=${1:-}

if [[ $CMREL = "" ]]; then
	echo "usage: $0 <path-to-cmrel>"
	exit 1
fi

logsfile=$(mktemp)

trap 'rm -f -- $logsfile' EXIT

BASE="validate-gomod-test/broken"

$CMREL --debug validate-gomod \
	--path $BASE \
	--no-dummy-modules example.com/nodummy \
	&>$logsfile && exitcode=$? || exitcode=$?

if [[ $exitcode -eq 0 ]]; then
	echo "ERROR: expected validate-gomod to fail but got a successful exit code"
	exit 1
fi

anyerrors=0

checkline() {
	rc=0

	grep -q "$1" $logsfile && rc=$? || rc=$?

	if [[ $rc -ne 0 ]]; then
		echo -e "ERROR: couldn't find required log line in output! wanted:\n > $1"
		anyerrors=1
	fi
}

checkline 'module "example.com/acmesolver" has Go version "1.18" but should have "1.19" to match core go.mod file'

checkline 'module "example.com/controller" replaces "example.org/somedependency" with "example.org/somedependency v1.1.2", but the expected replacement was "example.org/somedependency v1.0.1". All replaces should match the core go.mod file'

checkline 'module "example.com/cainjector" replaces "example.com/core" with "../../../ ", but the expected replacement was "../../ ". Core module replacements should point at the core module'

checkline 'module "example.com/cmctl" requires "example.org/somedependency" which is replaced by "example.org/somedependency v1.0.1" in the core module but is not replaced in this module. Submodules should have the same replacements as the core module'

checkline 'module "example.com/webhook" replaces "example.org/somedependency" with "../../../somedependency-local ", but the expected replacement was "example.org/somedependency v1.0.1". All replaces should match the core go.mod file'

checkline 'module "example.com/integration-tests" imports internal module "example.com/core" with incorrect version; should be "v0.0.0-00010101000000-000000000000"'

checkline 'module "example.com/e2e-tests" imports internal module "example.com/cmctl" with incorrect version; should be "v0.0.0-00010101000000-000000000000"'

checkline 'core module should have no local (filesystem) replaces, but has: "example.com/localreplace"'

checkline 'module "example.com/somebinary" requires the core module "example.com/core". The core module should have a filesystem replacement'

checkline 'module "example.com/nodummy" requires the core module "example.com/core". The core module should have a filesystem replacement'

if [[ $anyerrors -ne 0 ]]; then
	echo "+++ at least one error was found with validate-gomod output"
	echo "+++ full logs:"
	cat $logsfile
	exit 1
fi
