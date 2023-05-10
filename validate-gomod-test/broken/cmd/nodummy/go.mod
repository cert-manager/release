module example.com/nodummy

go 1.19

// This module should be in "NoDummyModules" allowing it to use an actual version for the core module
// It should fail because it "NoDummyModules" still requires a replace statement for the core module,
// which is missing here

require example.com/core v1.2.3
