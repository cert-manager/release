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

package gcb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/api/cloudbuild/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

const (
	Success = "SUCCESS"
	Failure = "FAILURE"
)

// LoadBuild will decode a cloudbuild.yaml file into a cloudbuild.Build
// structure and return it.
func LoadBuild(filename string) (*cloudbuild.Build, error) {
	f, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cb := cloudbuild.Build{}
	if err := yaml.UnmarshalStrict(f, &cb); err != nil {
		return nil, err
	}

	return &cb, nil
}

// SubmitBuild will submit a Build to the cloud build API.
// It will wait for the Create operation to complete, and then return an
// up-to-date copy of the Build from the server.
func SubmitBuild(svc *cloudbuild.Service, projectID string, build *cloudbuild.Build) (*cloudbuild.Build, error) {
	op, err := svc.Projects.Builds.Create(projectID, build).Do()
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: decoding build operation metadata")
	metadata := &cloudbuild.BuildOperationMetadata{}
	if err := json.Unmarshal(op.Metadata, metadata); err != nil {
		return nil, err
	}

	return metadata.Build, nil
}

// WaitForBuild will wait for the GCB Build with the given ID to complete
// before returning a final copy of the Build resource.
func WaitForBuild(svc *cloudbuild.Service, projectID string, id string) (*cloudbuild.Build, error) {
	var build *cloudbuild.Build
	var err error
	err = wait.PollInfinite(time.Second*5, func() (done bool, err error) {
		build, err = svc.Projects.Builds.Get(projectID, id).Do()
		if err != nil {
			return false, err
		}

		// TODO: invert this to check for Pending instead
		if build.Status == Success || build.Status == Failure {
			return true, nil
		}

		log.Printf("DEBUG: build %q still in progress...", build.Id)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return build, err
}

// ListBuildsWithTag will list all Builds that have the given tag value set,
// paginating through any responses from the GCB API that use pagination.
func ListBuildsWithTag(ctx context.Context, svc *cloudbuild.Service, projectID string, tag string) ([]*cloudbuild.Build, error) {
	var builds []*cloudbuild.Build
	if err := svc.Projects.Builds.List(projectID).Filter("tags="+tag).Pages(ctx, func(resp *cloudbuild.ListBuildsResponse) error {
		builds = append(builds, resp.Builds...)
		return nil
	}); err != nil {
		return nil, err
	}

	return builds, nil
}

// TagForReleaseVersion will return a tag that should be added to Builds for
// a given releaseVersion/gitRef pair.
// This is used to discover existing GCB builds for a release when running the
// stage command.
func TagForReleaseVersion(releaseVersion, gitRef string) string {
	return fmt.Sprintf("%s-%s", releaseVersion, gitRef)
}

// NewestGreenBuild will find the newest passing release build for a given
// release version and git commit ref.
func NewestGreenBuild(ctx context.Context, svc *cloudbuild.Service, projectID, releaseVersion, gitRef string) *cloudbuild.Build {
	return nil
}
