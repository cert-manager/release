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
	"io"
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

// MetadataVerifier verifies that the raw metadata.json bytes are authentic,
// given the detached signature bytes read from alongside them.
type MetadataVerifier func(ctx context.Context, metadata, signature []byte) error

// NewStagedRelease loads and returns a staged release from the given GCS objects.
func NewStagedRelease(ctx context.Context, name, prefix string, verify MetadataVerifier, objects ...*storage.ObjectHandle) (*Staged, error) {
	meta, metaBytes, err := loadReleaseMetadataFile(ctx, objects...)
	if err != nil {
		return nil, err
	}

	if verify != nil {
		// The signature is loaded on a best-effort basis: it may legitimately be
		// absent, in which case verify is handed an empty signature and decides
		// what to do about it.
		signature, err := loadReleaseMetadataSignature(ctx, objects...)
		if err != nil {
			return nil, err
		}

		if err := verify(ctx, metaBytes, signature); err != nil {
			return nil, err
		}
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

// loadReleaseMetadataFile locates metadata.json amongst the staged objects,
// decodes it, and returns both the parsed Metadata and the exact bytes it was decoded from.
func loadReleaseMetadataFile(ctx context.Context, objs ...*storage.ObjectHandle) (*Metadata, []byte, error) {
	metadataObj := findObjectByBaseName(MetadataFileName, objs...)
	if metadataObj == nil {
		return nil, nil, fmt.Errorf("release metadata not found")
	}

	body, err := readObject(ctx, metadataObj)
	if err != nil {
		return nil, nil, err
	}

	var m Metadata
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, nil, err
	}

	return &m, body, nil
}

// loadReleaseMetadataSignature locates the metadata.json.sig object amongst the
// staged objects and returns its contents.
func loadReleaseMetadataSignature(ctx context.Context, objs ...*storage.ObjectHandle) ([]byte, error) {
	signatureObj := findObjectByBaseName(MetadataSignatureFileName, objs...)
	if signatureObj == nil {
		return nil, nil
	}

	return readObject(ctx, signatureObj)
}

// findObjectByBaseName returns the first object whose path has the given base
// name, or nil if none match.
func findObjectByBaseName(baseName string, objs ...*storage.ObjectHandle) *storage.ObjectHandle {
	for _, f := range objs {
		if filepath.Base(f.ObjectName()) == baseName {
			return f
		}
	}
	return nil
}

// readObject reads the full contents of a GCS object into memory.
func readObject(ctx context.Context, obj *storage.ObjectHandle) ([]byte, error) {
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
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
