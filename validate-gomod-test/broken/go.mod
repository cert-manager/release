module example.com/core

go 1.19

// We have localreplace replaced below intentionally, which should error as the core
// module should have no filesystem replaces

replace (
	example.com/localreplace => ../../
	example.org/somedependency v1.1.1 => example.org/somedependency v1.0.1
)

require (
	example.com/localreplace v0.0.1
	example.org/somedependency v1.1.1
	example.org/someotherdependency v1.5.5
)
