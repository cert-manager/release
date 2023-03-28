module example.com/webhook

go 1.19

// somedependency is replaced by something on the filesystem below, which should
// fail as the core module has an explicit replace for it and submodules should
// match the core module

replace (
	example.com/core => ../../
	example.org/somedependency v1.1.1 => ../../../somedependency-local
)

require (
	example.com/core v0.0.0-00010101000000-000000000000
	example.org/somedependency v1.1.1
)
