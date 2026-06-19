module example.com/acmesolver

// The go version here is intentionally incorrect. This should error.

go 1.18

replace example.com/core => ../../

require example.com/core v0.0.0-00010101000000-000000000000
