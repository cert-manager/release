module example.com/cmctl

go 1.21.3

// somedependency is not replaced here but is required, which should fail
// as the core module replaces somedependency and that replacement should be
// uniform across all submodules

replace example.com/core => ../../

require (
	example.com/core v0.0.0-00010101000000-000000000000
	example.org/somedependency v1.1.1
)
