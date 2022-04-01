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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
)

// Staged is a release build staged in a GCS bucket.
// It provides convenience methods to interact with release build and inspect
// metadata.
type Staged struct {
	name      string
	prefix    string
	meta      Metadata
	artifacts []StagedArtifact
}

// StagedArtifact represents a single artifact within a release, with some
// associated metadata read from the release metadata.json file.
type StagedArtifact struct {
	Metadata     ArtifactMetadata
	ObjectHandle *storage.ObjectHandle
}

func NewStagedRelease(ctx context.Context, name, prefix string, objects ...*storage.ObjectHandle) (*Staged, error) {
	meta, err := loadReleaseMetadataFile(ctx, objects...)
	if err != nil {
		return nil, err
	}

	artifacts, err := crossReferenceArtifactMetadata(*meta, name, prefix, objects...)
	if err != nil {
		return nil, err
	}

	return &Staged{
		name:      name,
		prefix:    prefix,
		meta:      *meta,
		artifacts: artifacts,
	}, nil
}

// Name will return the name of the release in the GCS bucket
func (s Staged) Name() string {
	return s.name
}

// Metadata will return metadata information about the release.
func (s Staged) Metadata() Metadata {
	return s.meta
}

// ArtifactsOfKind returns a list of staged artifacts of the type denoted by
// `kind`. A kind may be 'server', 'manifests', 'test' etc.
func (s Staged) ArtifactsOfKind(kind string) []StagedArtifact {
	var objs []StagedArtifact
	for _, obj := range s.artifacts {
		kindPrefix := releaseObjectPrefix + kind
		if strings.HasPrefix(obj.Metadata.Name, kindPrefix) {
			objs = append(objs, obj)
		}
	}
	return objs
}

func loadReleaseMetadataFile(ctx context.Context, objs ...*storage.ObjectHandle) (*Metadata, error) {
	var metadataObj *storage.ObjectHandle
	for _, f := range objs {
		if filepath.Base(f.ObjectName()) == MetadataFileName {
			metadataObj = f
			break
		}
	}

	if metadataObj == nil {
		return nil, fmt.Errorf("release metadata not found")
	}

	r, err := metadataObj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var m Metadata
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

func crossReferenceArtifactMetadata(meta Metadata, name, prefix string, objs ...*storage.ObjectHandle) ([]StagedArtifact, error) {
	var artifacts []StagedArtifact
	objectMap := mapifyObjectHandles(objs...)
	objPrefix := prefix + name + "/"
	for _, a := range meta.Artifacts {
		obj, ok := objectMap[objPrefix+a.Name]
		if !ok {
			return nil, fmt.Errorf("artifact %q named in manifest file but not present in list of GCS objects (path tested: %s)", a.Name, objPrefix+a.Name)
		}
		artifacts = append(artifacts, StagedArtifact{
			Metadata:     a,
			ObjectHandle: obj,
		})
	}
	return artifacts, nil
}

func mapifyObjectHandles(objs ...*storage.ObjectHandle) map[string]*storage.ObjectHandle {
	m := make(map[string]*storage.ObjectHandle, len(objs))
	for _, obj := range objs {
		m[obj.ObjectName()] = obj
	}
	return m
}

const (
	// The prefix used to identify release artifact objects.
	releaseObjectPrefix = "cert-manager-"
)
