module example.com/core

go 1.19

replace example.org/somedependency v1.1.1 => example.org/somedependency v1.0.1

require (
	example.org/adep v0.0.1
	example.org/somedependency v1.1.1
	example.org/someotherdependency v1.5.5
)
