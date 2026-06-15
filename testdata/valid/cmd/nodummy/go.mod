module example.com/nodummy

go 1.21.3

// This module should be in "NoDummyModules" allowing it to use an actual version for the core module,
// but still requires a replace for the core module.

replace example.com/core => ../../

require example.com/core v1.2.3
