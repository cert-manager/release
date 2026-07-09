/*
Copyright 2026 The cert-manager Authors.

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

package cmd

import (
	"runtime/debug"
	"testing"
)

func TestReleaseRepoRefFromBuildInfo(t *testing.T) {
	const commit = "f6da9c76877551ef32503b17189bb178501f59a7"

	tests := []struct {
		name    string
		info    *debug.BuildInfo
		ok      bool
		want    string
		wantErr bool
	}{
		{
			name:    "no build info",
			info:    nil,
			ok:      false,
			wantErr: true,
		},
		{
			name: "clean working tree pins to the commit",
			ok:   true,
			info: &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: commit},
				{Key: "vcs.modified", Value: "false"},
			}},
			want: commit,
		},
		{
			name: "modified working tree fails closed",
			ok:   true,
			info: &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: commit},
				{Key: "vcs.modified", Value: "true"},
			}},
			wantErr: true,
		},
		{
			name: "module install pins to the version",
			ok:   true,
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20230101000000-" + commit[:12]}},
			want: "v0.0.0-20230101000000-" + commit[:12],
		},
		{
			name:    "devel build without vcs stamping fails closed",
			ok:      true,
			info:    &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			wantErr: true,
		},
		{
			name:    "no revision and no version fails closed",
			ok:      true,
			info:    &debug.BuildInfo{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := releaseRepoRefFromBuildInfo(tt.info, tt.ok)
			if (err != nil) != tt.wantErr {
				t.Fatalf("releaseRepoRefFromBuildInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("releaseRepoRefFromBuildInfo() = %q, want %q", got, tt.want)
			}
		})
	}
}
