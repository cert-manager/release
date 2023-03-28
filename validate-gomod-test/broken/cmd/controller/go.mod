module example.com/controller

go 1.19

// The replacement for example.org/somedependency below is intentionally to a different version (v1.1.2)
// than the one in the core go.mod file (which uses v1.0.1)
// We expect this to error

replace (
	example.com/core => ../../
	example.org/somedependency v1.1.1 => example.org/somedependency v1.1.2
)

require (
	example.com/core v0.0.0-00010101000000-000000000000
	example.org/somedependency v1.1.1
)
