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

package helm

import (
	"context"

	"github.com/cert-manager/release/pkg/shell"
)

// PushChartToOCI pushes a Helm chart to an OCI registry using the helm command.
// The helm command automatically pushes the .prov file if it exists alongside the chart.
func PushChartToOCI(ctx context.Context, chartPath, ociURL, helmPath string) error {
	return shell.Command(ctx, "", helmPath, "push", chartPath, ociURL)
}

// CopyChartTag copies a chart from one OCI tag to another using crane.
func CopyChartTag(ctx context.Context, sourceTag, destTag, cranePath string) error {
	return shell.Command(ctx, "", cranePath, "copy", sourceTag, destTag)
}
