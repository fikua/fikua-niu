//go:build !race

package integration

// raceEnabled is false in normal (non -race) test builds. See
// race_detector.go for the true case.
const raceEnabled = false
