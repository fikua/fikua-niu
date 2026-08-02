//go:build race

package integration

// raceEnabled is true only in binaries built with `go test -race`. Used
// by auth_perf_test.go to skip a timing assertion that -race's
// instrumentation overhead makes meaningless, without silently loosening
// the budget for normal (non-race) runs — see race_detector_off.go for
// the false case.
const raceEnabled = true
