package testutil

import (
	"fmt"
	"os"
	"testing"
)

// so tests pass if given a -run that doesn't include TestSample
var ranSample = false

func TestMain(m *testing.M) {
	m.Run()
	isLeaked := CheckLeakedGoroutine()
	if ranSample && !isLeaked {
		fmt.Fprintln(os.Stderr, "expected leaky goroutines but none is detected")
		os.Exit(1)
	}
	os.Exit(0)
}

func TestSample(t *testing.T) {
	defer AfterTest(t)
	ranSample = true
	for range make([]struct{}, 100) {
		go func() {
			select {}
		}()
	}
}
