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

package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const (
	validateGoModCommand         = "validate-gomod"
	validateGoModDescription     = "Parse cert-manager go.mod files to ensure they're importable"
	validateGoModLongDescription = `Parses cert-manager go.mod files to enforce certain requirements on them.

NB: We talk about "core" referring to the main go.mod file for the cert-manager
packge, and "sub" referring to go.mod files for modules we don't expect people
to import.

We use the term "internal module" to refer to all modules within the repo.

Ensures that:

- Any replace directives for 3rd party dependencies in core are also present
  used for all subs which also have those dependencies, preventing drift
- All subs use an invalid version of all internal modules in their go.mod so they're
  forced to rely on replace directives pointing to the local module in the repo
- All modules declare the same version of Golang`

	// dummyCoreImportVersion is the expected version string for any import of the core module.
	// This dummy string makes it clearer that the module is replaced with a local filesystem
	// version, and means that anyone (incorrectly) importing a submodule will see an error about
	// an incorrect version of cert-manager
	dummyCoreImportVersion = "v0.0.0-00010101000000-000000000000"

	// hardLocalReplace is the path from submodules to the root of the repo. This won't
	// necessarily always be hardcoded, but for now hardcoding it works
	hardLocalReplace = "../../"
)

var (
	validateGoModExample = fmt.Sprintf(`To validate a local checkout:

%s %s --path <path-to-checkout>`, rootCommand, validateGoModCommand)
)

type validateGoModOptions struct {
	// Path is the path of the cert-manager checkout to validate
	Path string
}

func (o *validateGoModOptions) AddFlags(fs *flag.FlagSet, markRequired func(string)) {
	fs.StringVar(&o.Path, "path", "", "Path of cert-manager checkout to validate")

	markRequired("path")
}

func (o *validateGoModOptions) print() {
	log.Printf("%s options:", validateGoModCommand)
	log.Printf("  Path: %q", o.Path)
}

func validateGoModCmd(rootOpts *rootOptions) *cobra.Command {
	o := &validateGoModOptions{}

	cmd := &cobra.Command{
		Use:          validateGoModCommand,
		Short:        validateGoModDescription,
		Long:         validateGoModLongDescription,
		Example:      validateGoModExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			o.print()
			log.Printf("---")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidateGoMod(rootOpts, o)
		},
	}

	o.AddFlags(cmd.Flags(), mustMarkRequired(cmd.MarkFlagRequired))

	return cmd
}

func runValidateGoMod(rootOpts *rootOptions, o *validateGoModOptions) error {
	var validationErrors []error

	allInternalModules, err := findInternalModules(o.Path)
	if err != nil {
		return fmt.Errorf("failed to find all submodules in %q: %s", o.Path, err.Error())
	}

	if errs := allInternalModules.checkReplaces(); len(errs) > 0 {
		validationErrors = append(validationErrors, errs...)
	}

	if err := allInternalModules.checkCoreModuleReplacements(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if errs := allInternalModules.checkInternalModuleRequirements(); len(errs) > 0 {
		validationErrors = append(validationErrors, errs...)
	}

	if errs := allInternalModules.checkGoVersions(); len(errs) > 0 {
		validationErrors = append(validationErrors, errs...)
	}

	if len(validationErrors) > 0 {
		log.Println("validation failed! errors:")
		for _, err := range validationErrors {
			log.Printf("  %s", err.Error())
		}

		return fmt.Errorf("validation failed")
	}

	return nil
}

type internalModuleList struct {
	modules []*internalModule

	coreModule *internalModule
	submodules []*internalModule

	replaceMap map[string]module.Version

	// internalModuleNames is used for quickly answering the question "is this module name for an internal module". A map is used for O(1) lookup; the boolean is ignored
	internalModuleNames map[string]bool
}

type internalModule struct {
	Name string

	// LocalRepoPath is the path to the go.mod file relative to the root of the repository
	// So if the module "example.com/asd" is in "/home/user/repo/cmd/xyz/go.mod",
	// then LocalRepoPath would be "cmd/xyz/"
	LocalRepoPath string

	// FullGoModPath is the absolute path of the go.mod file for this submodule
	// So if the module "example.com/asd" is in "/home/user/repo/cmd/xyz/go.mod",
	// then FullGoModPath would be "/home/user/repo/cmd/xyz/go.mod"
	FullGoModPath string

	Module *modfile.File
}

func findInternalModules(root string) (*internalModuleList, error) {
	var iml internalModuleList

	iml.internalModuleNames = make(map[string]bool)

	coreModulePath := filepath.Join(root, "go.mod")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		base := filepath.Base(path)

		if d.IsDir() {
			if base == "bin" || base == "_bin" || strings.HasPrefix(base, ".") {
				return fs.SkipDir
			}

			return nil
		}

		if base != "go.mod" {
			return nil
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read module file %q: %s", path, err.Error())
		}

		f, err := modfile.Parse(path, contents, nil)
		if err != nil {
			return fmt.Errorf("failed to parse module file %q: %s", path, err.Error())
		}

		newMod := internalModule{
			Name:          f.Module.Mod.Path,
			LocalRepoPath: strings.TrimPrefix(path, root+"/"),
			FullGoModPath: path,
			Module:        f,
		}

		iml.modules = append(iml.modules, &newMod)
		iml.internalModuleNames[newMod.Name] = true

		if path == coreModulePath {
			iml.coreModule = &newMod
		} else {
			iml.submodules = append(iml.submodules, &newMod)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if iml.coreModule == nil {
		return nil, fmt.Errorf("couldn't find and load core module from expected path %q", coreModulePath)
	}

	iml.replaceMap = make(map[string]module.Version)

	// all core module replacements of third party depedencies should be duplicated for any
	// submodule, so that the core module is the single source of truth
	for _, replaceStmt := range iml.coreModule.Module.Replace {
		iml.replaceMap[replaceStmt.Old.Path] = replaceStmt.New
	}

	// all submodules should be replaced with local filesystem replacements, so that everything
	// only builds against what's in the repo
	for _, submodule := range iml.submodules {
		iml.replaceMap[submodule.Module.Module.Mod.Path] = module.Version{
			Path:    filepath.Join(hardLocalReplace, strings.TrimSuffix(submodule.LocalRepoPath, "go.mod")) + "/",
			Version: "",
		}
	}

	// the core module should always be replaced with a filesystem replacement
	iml.replaceMap[iml.coreModule.Module.Module.Mod.Path] = module.Version{
		Path:    hardLocalReplace,
		Version: "",
	}

	return &iml, nil
}

// checkCoreModuleReplacements ensures that the core module doesn't have any local replacements which
// would break third party importers of that module
func (iml *internalModuleList) checkCoreModuleReplacements() error {
	var localReplaces []string

	for _, replaceStmt := range iml.coreModule.Module.Replace {
		if replaceStmt.New.Version == "" {
			localReplaces = append(localReplaces, replaceStmt.Old.Path)
		}
	}

	if len(localReplaces) > 0 {
		return fmt.Errorf("core module should have no local (filesystem) replaces, but has: %q", strings.Join(localReplaces, ", "))
	}

	return nil
}

// checkReplaces verifies that all internal modules have valid replace statements, meaning that:
// - if a replace statement is for an internal module, it's using a filesystem replacement
// - internal modules require a hardcoded dummy version
// - if a replace statement is defined in the core module, then any submodules have the same replacement
func (iml *internalModuleList) checkReplaces() []error {
	var errs []error

	for _, m := range iml.modules {
		foundReplaces := make(map[string]bool)

		for _, replaceStmt := range m.Module.Replace {
			expectedReplace, exists := iml.replaceMap[replaceStmt.Old.Path]
			if !exists {
				// It's fine if we have an extra replacement in a submodule which the core module doesn't have
				continue
			}

			foundReplaces[replaceStmt.Old.Path] = true

			if replaceStmt.New.Path != expectedReplace.Path || replaceStmt.New.Version != expectedReplace.Version {
				errs = append(errs, fmt.Errorf("module %q replaces %q with \"%s %s\", but the expected replacement was \"%s %s\". All replaces should match the core go.mod file and all internal modules should have local replacements", m.Name, replaceStmt.Old.Path, replaceStmt.New.Path, replaceStmt.New.Version, expectedReplace.Path, expectedReplace.Version))
				continue
			}
		}

		for _, requireStmt := range m.Module.Require {
			coreReplace, shouldReplace := iml.replaceMap[requireStmt.Mod.Path]
			if !shouldReplace {
				continue
			}

			_, wasReplaced := foundReplaces[requireStmt.Mod.Path]
			if !wasReplaced {
				errs = append(errs, fmt.Errorf("module %q requires %q which is replaced by \"%s %s\" in the core module but is not replaced in this module. Submodules should have the same replacements as the core module", m.Name, requireStmt.Mod.Path, coreReplace.Path, coreReplace.Version))
			}
		}
	}

	return errs
}

// checkInternalModuleRequirements ensures that every internal module uses a dummy version when requiring other internal modules.
func (iml *internalModuleList) checkInternalModuleRequirements() []error {
	var errs []error

	for _, m := range iml.modules {
		for _, requireStmt := range m.Module.Require {
			_, isInternal := iml.internalModuleNames[requireStmt.Mod.Path]
			if !isInternal {
				continue
			}

			if requireStmt.Mod.Version != dummyCoreImportVersion {
				errs = append(errs, fmt.Errorf("module %q imports internal module %q with incorrect version; should be %q", m.Name, requireStmt.Mod.Path, dummyCoreImportVersion))
			}
		}
	}

	return errs
}

// checkGoVersions ensures that all internal modules use the same version of Go
func (iml *internalModuleList) checkGoVersions() []error {
	coreGoVersion := iml.coreModule.Module.Go.Version

	var errs []error

	for _, s := range iml.submodules {
		if s.Module.Go.Version != coreGoVersion {
			errs = append(errs, fmt.Errorf("module %q has Go version %q but should have %q to match core go.mod file", s.Name, s.Module.Go.Version, coreGoVersion))
		}
	}

	return errs
}
