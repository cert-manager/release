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

.PHONY: build
build: bin/cmrel

.PHONY: bin/cmrel
bin/cmrel:
	go build -o $@ ./cmd/cmrel

.PHONY: presubmit
presubmit: bin/cmrel test verify-boilerplate
	./test/presubmit.sh $<

.PHONY: verify-boilerplate
verify-boilerplate:
	@./hack/verify_boilerplate.py

.PHONY: clean
clean:
	rm -rf ./bin ./cmrel

.PHONY: test
test: test-validate-gomod test-validate-gomod-success
	@# TODO: this should be go test ./... but one of the tests was broken a while back and needs fixing first
	go test ./cmd/cmrel/cmd/...
	go test ./pkg/release
	go test ./pkg/release/helm
	go test ./pkg/release/manifests
	@# go test ./pkg/release/validation  # this one is broken
	go test ./pkg/sign

.PHONY: test-validate-gomod
test-validate-gomod: bin/cmrel
	./hack/test-validate-gomod.sh $<

.PHONY: test-validate-gomod-success
test-validate-gomod-success: bin/cmrel
	./hack/test-validate-gomod-success.sh $<
