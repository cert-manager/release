module example.com/cainjector

go 1.19

// This replace is intentionally incorrect and should error.
replace example.com/core => ../../../

require example.com/core v0.0.0-00010101000000-000000000000