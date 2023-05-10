module example.com/directimport

go 1.19

// This module should be passed to DirectImportModules and so
// shouldn't need a replace statement and should be able to use an actual
// version for the core module

require example.com/core v1.2.3
