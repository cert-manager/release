/*
Copyright 2022 The cert-manager Authors.

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

package testgen

type TestConfigurer func(*Test)

// testTemplate defines a 'default' test, where standard parameters can be set. All tests
// should have a name and a friendly description of what they do.
// Callers can also pass a list of "configurers" which will modify the template before
// it's returned for use.
func testTemplate(name string, description string, configurers ...TestConfigurer) *Test {
	test := &Test{
		Name:     name,
		Agent:    "kubernetes",
		Decorate: true,
		Annotations: map[string]string{
			"description": description,
		},
		Labels: make(map[string]string),
		Spec: TestSpec{
			DNSConfig: DefaultDNSConfig(),
		},
	}

	for _, c := range configurers {
		c(test)
	}

	return test
}

func addMakeVolumesLabel(test *Test) {
	test.Labels["preset-make-volumes"] = "true"
}

func addServiceAccountLabel(test *Test) {
	test.Labels["preset-service-account"] = "true"
}

func addDindLabel(test *Test) {
	test.Labels["preset-dind-enabled"] = "true"
}

func addCloudflareCredentialsLabel(test *Test) {
	test.Labels["preset-cloudflare-credentials"] = "true"
}

func addRetryFlakesLabel(test *Test) {
	test.Labels["preset-retry-flakey-tests"] = "true"
}

func addDefaultE2EVolumeLabels(test *Test) {
	test.Labels["preset-default-e2e-volumes"] = "true"
}

func addGinkgoSkipDefaultLabel(test *Test) {
	test.Labels["preset-ginkgo-skip-default"] = "true"
}

func addDisableFeatureGatesLabel(test *Test) {
	test.Labels["preset-disable-all-alpha-beta-feature-gates"] = "true"
}

func addVenafiTPPLabels(test *Test) {
	test.Labels["preset-ginkgo-focus-venafi-tpp"] = "true"
	test.Labels["preset-venafi-tpp-credentials"] = "true"
}

func addVenafiBothLabels(test *Test) {
	test.Labels["preset-ginkgo-focus-venafi"] = "true"

	test.Labels["preset-venafi-cloud-credentials"] = "true"
	test.Labels["preset-venafi-tpp-credentials"] = "true"
}

func addVenafiCloudLabels(test *Test) {
	test.Labels["preset-ginkgo-focus-venafi-cloud"] = "true"
	test.Labels["preset-venafi-cloud-credentials"] = "true"
}

func addStandardE2ELabels(kubernetesVersion string) TestConfigurer {
	return func(test *Test) {
		addDefaultE2EVolumeLabels(test)
		addGinkgoSkipDefaultLabel(test)

		majorVersion, minorVersion, err := splitKubernetesVersion(kubernetesVersion)
		if err != nil {
			// note: we panic here because this tool is developer-facing and because an
			// error here suggests programmer error (e.g. a typo'd k8s version)
			// adding 'return nil' for every configurer - most of which shouldn't fail in
			// any reasonable scenario - seems far messier than a panic here
			panic(err)
		}

		if majorVersion == 1 && minorVersion < 22 {
			// SSA (server-side apply) is only fully supported in k8s 1.22+
			test.Labels["preset-enable-all-feature-gates-disable-ssa"] = "true"
			return
		}

		test.Labels["preset-enable-all-feature-gates"] = "true"
	}
}

func addTestGridAnnotations(dashboardName string) TestConfigurer {
	return func(test *Test) {
		test.Annotations["testgrid-create-test-group"] = "true"
		test.Annotations["testgrid-dashboards"] = dashboardName
		test.Annotations["testgrid-alert-email"] = AlertEmailAddress
	}
}

func addMaxConcurrency(maxConcurrency int) TestConfigurer {
	return func(test *Test) {
		test.MaxConcurrency = maxConcurrency
	}
}
