module example.com/integration-tests

go 1.21.3

replace example.com/core => ../../

replace example.com/cmctl => ../../cmd/ctl/

// core below erroneously has an actual version. This should error.

require (
	example.com/cmctl v0.0.0-00010101000000-000000000000
	example.com/core v1.11.0
)
