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

package release

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type Bucket struct {
	bucket *storage.BucketHandle
	prefix string
}

func NewBucket(bucket *storage.BucketHandle, prefix, releaseType string) *Bucket {
	return &Bucket{bucket: bucket, prefix: fmt.Sprintf("%s/%s/", prefix, releaseType)}
}

// GetRelease will fetch a single release from the bucket with the given name.
// A release's name is the name of the directory the metadata.json file for is
// the release is contained within.
func (b *Bucket) GetRelease(ctx context.Context, name string) (*Staged, error) {
	queryPath := b.prefix + name + "/"
	stagedReleases := map[string][]*storage.ObjectHandle{}
	objs := b.bucket.Objects(ctx, &storage.Query{Prefix: queryPath})
	for {
		objAttr, err := objs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		obj := b.bucket.Object(objAttr.Name)
		releaseName := NameForObjectPath(obj.ObjectName(), b.prefix)
		stagedReleases[releaseName] = append(stagedReleases[releaseName], obj)
	}
	if len(stagedReleases) > 1 {
		return nil, fmt.Errorf("internal error getting release: multiple releases found")
	}
	// iterate over the map. There is at most one element so return in the loop
	for name, objs := range stagedReleases {
		rel, err := NewStagedRelease(ctx, name, b.prefix, objs...)
		if err != nil {
			return nil, fmt.Errorf("failed to load staged release: %w", err)
		}
		return rel, nil
	}
	return nil, fmt.Errorf("no release found in path %q", queryPath)
}

// ListReleases will list releases in a bucket.
// If 'version' is provided, the list will be filtered to only releases with
// the specified version.
// If 'version' AND 'gitRef' are provided, the list will be filtered to only
// releases with the specified version built at the specified commit ref.
// Specifying 'gitRef' without 'version' is not supported.
func (b *Bucket) ListReleases(ctx context.Context, version, gitRef string) ([]Staged, error) {
	stagedReleases := map[string][]*storage.ObjectHandle{}
	objs := b.bucket.Objects(ctx, &storage.Query{Prefix: b.prefix + pathSuffixForVersion(version, gitRef)})
	for {
		objAttr, err := objs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		obj := b.bucket.Object(objAttr.Name)
		releaseName := NameForObjectPath(obj.ObjectName(), b.prefix)
		stagedReleases[releaseName] = append(stagedReleases[releaseName], obj)
	}
	var staged []Staged
	for name, objs := range stagedReleases {
		rel, err := NewStagedRelease(ctx, name, b.prefix, objs...)
		if err != nil {
			log.Printf("failed to load staged release: %v", err)
			continue
		}
		staged = append(staged, *rel)
	}
	return staged, nil
}

// NameForObjectPath will return the name of the release that a given object
// path is a member of by inspecting the path and trimming the prefix.
func NameForObjectPath(path, prefix string) string {
	trimmedPath := strings.TrimPrefix(path, prefix)
	releaseName := strings.Split(trimmedPath, "/")[0]
	return releaseName
}

func pathSuffixForVersion(version, gitRef string) string {
	if version == "" {
		return ""
	}
	return version + "-" + gitRef
}
