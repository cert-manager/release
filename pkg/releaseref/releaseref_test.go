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

package releaseref

import (
	"runtime/debug"
	"testing"
)

func TestFromBuildInfo(t *testing.T) {
	const commit = "f6da9c76877551ef32503b17189bb178501f59a7"

	tests := []struct {
		name    string
		info    *debug.BuildInfo
		want    string
		wantErr bool
	}{
		{
			name: "clean working tree pins to the commit",
			info: &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: commit},
				{Key: "vcs.modified", Value: "false"},
			}},
			want: commit,
		},
		{
			name: "modified working tree fails closed",
			info: &debug.BuildInfo{Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: commit},
				{Key: "vcs.modified", Value: "true"},
			}},
			wantErr: true,
		},
		{
			name: "module install pins to the version",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20230101000000-" + commit[:12]}},
			want: "v0.0.0-20230101000000-" + commit[:12],
		},
		{
			name:    "devel build without vcs stamping fails closed",
			info:    &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			wantErr: true,
		},
		{
			name:    "no revision and no version fails closed",
			info:    &debug.BuildInfo{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fromBuildInfo(tt.info)
			if (err != nil) != tt.wantErr {
				t.Fatalf("fromBuildInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("fromBuildInfo() = %q, want %q", got, tt.want)
			}
		})
	}
}
