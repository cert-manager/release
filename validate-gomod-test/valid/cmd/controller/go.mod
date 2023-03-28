module example.com/controller

go 1.19

replace (
	example.com/core => ../../
	example.org/somedependency v1.1.1 => example.org/somedependency v1.0.1
)

require (
	example.com/core v0.0.0-00010101000000-000000000000
	example.org/somedependency v1.1.1
)
