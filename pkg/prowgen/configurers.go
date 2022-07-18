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

package prowgen

type JobConfigurer func(*Job)

// jobTemplate defines a 'default' job, where standard parameters can be set. All jobs
// should have a name and a friendly description of what they do.
// Callers can also pass a list of "configurers" which will modify the template before
// it's returned for use.
func jobTemplate(name string, description string, configurers ...JobConfigurer) *Job {
	job := &Job{
		Name:     name,
		Agent:    "kubernetes",
		Decorate: true,
		Annotations: map[string]string{
			"description": description,
		},
		Labels: make(map[string]string),
		Spec: JobSpec{
			DNSConfig: DefaultDNSConfig(),
		},
	}

	for _, c := range configurers {
		c(job)
	}

	return job
}

func addMakeVolumesLabel(job *Job) {
	job.Labels["preset-make-volumes"] = "true"
}

func addServiceAccountLabel(job *Job) {
	job.Labels["preset-service-account"] = "true"
}

func addDindLabel(job *Job) {
	job.Labels["preset-dind-enabled"] = "true"
}

func addCloudflareCredentialsLabel(job *Job) {
	job.Labels["preset-cloudflare-credentials"] = "true"
}

func addRetryFlakesLabel(job *Job) {
	job.Labels["preset-retry-flakey-jobs"] = "true"
}

func addDefaultE2EVolumeLabels(job *Job) {
	job.Labels["preset-default-e2e-volumes"] = "true"
}

func addGinkgoSkipDefaultLabel(job *Job) {
	job.Labels["preset-ginkgo-skip-default"] = "true"
}

func addDisableFeatureGatesLabel(job *Job) {
	job.Labels["preset-disable-all-alpha-beta-feature-gates"] = "true"
}

func addVenafiTPPLabels(job *Job) {
	job.Labels["preset-ginkgo-focus-venafi-tpp"] = "true"
	job.Labels["preset-venafi-tpp-credentials"] = "true"
}

func addVenafiBothLabels(job *Job) {
	job.Labels["preset-ginkgo-focus-venafi"] = "true"

	job.Labels["preset-venafi-cloud-credentials"] = "true"
	job.Labels["preset-venafi-tpp-credentials"] = "true"
}

func addVenafiCloudLabels(job *Job) {
	job.Labels["preset-ginkgo-focus-venafi-cloud"] = "true"
	job.Labels["preset-venafi-cloud-credentials"] = "true"
}

func addStandardE2ELabels(kubernetesVersion string) JobConfigurer {
	return func(job *Job) {
		addDefaultE2EVolumeLabels(job)
		addGinkgoSkipDefaultLabel(job)

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
			job.Labels["preset-enable-all-feature-gates-disable-ssa"] = "true"
			return
		}

		job.Labels["preset-enable-all-feature-gates"] = "true"
	}
}

func addTestGridAnnotations(dashboardName string) JobConfigurer {
	return func(job *Job) {
		job.Annotations["testgrid-create-job-group"] = "true"
		job.Annotations["testgrid-dashboards"] = dashboardName
		job.Annotations["testgrid-alert-email"] = AlertEmailAddress
	}
}

func addMaxConcurrency(maxConcurrency int) JobConfigurer {
	return func(job *Job) {
		job.MaxConcurrency = maxConcurrency
	}
}
