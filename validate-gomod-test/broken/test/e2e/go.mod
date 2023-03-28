module example.com/e2e-tests

go 1.19

// cmctl has an actual version below, which should fail as we expect all binaries and tests
// to be built against the current checkout and not to use upstream versions

replace example.com/core => ../../

require (
	example.com/cmctl v1.11.0
	example.com/core v0.0.0-00010101000000-000000000000
)
