module example.com/integration-tests

go 1.19

replace example.com/core => ../../

replace example.com/cmctl => ../../cmd/ctl/

require (
	example.com/cmctl v0.0.0-00010101000000-000000000000
	example.com/core v0.0.0-00010101000000-000000000000
)
