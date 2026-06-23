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

package shell

import (
	"context"
	"os"
	"os/exec"
)

// Runner runs a command. Tests can substitute fake implementations to record
// invocations without actually executing the binary.
type Runner func(ctx context.Context, workDir string, cmd string, args ...string) error

// Default is the Runner that actually invokes the command via exec.CommandContext,
// streaming stdout/stderr to the process's standard streams.
var Default Runner = func(ctx context.Context, workDir string, cmd string, args ...string) error {
	c := exec.CommandContext(ctx, cmd, args...)

	// redirect all output
	// TODO: honour --debug flag
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	c.Dir = workDir

	return c.Run()
}

// Command runs the given command with the given args using the Default runner.
func Command(ctx context.Context, workDir string, cmd string, args ...string) error {
	return Default(ctx, workDir, cmd, args...)
}
