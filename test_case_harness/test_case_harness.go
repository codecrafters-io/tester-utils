package test_case_harness

import (
	"github.com/codecrafters-io/tester-utils/executable"
	"github.com/codecrafters-io/tester-utils/logger"
	"github.com/codecrafters-io/tester-utils/tester_cache"
)

// TestCaseHarness is passed to your TestCase's TestFunc.
//
// If the program is a long-lived program that must be alive during the duration of the test (like a Redis server),
// do something like this at the start of your test function:
//
//	if err := harness.Executable.Start(); err != nil {
//	   return err
//	}
//	harness.RegisterTeardownFunc(func() { harness.Executable.Kill() })
//
// If the program is a script that must be executed and then checked for output (like a Git command), use it like this:
//
//	result, err := harness.Executable.Run("cat-file", "-p", "sha")
//	if err != nil {
//	    return err
//	 }
type TestCaseHarness struct {
	// Logger is to be used for all logs generated from the test function.
	Logger *logger.Logger

	// Executable is the program to be tested.
	Executable *executable.Executable

	// teardownFuncs are run once the error has been reported to the user
	teardownFuncs []func()

	// TesterCache can be used as a key-value cache by the tester in test functions
	// for faster access of values (eg. bypassing redundant network requests)
	TesterCache *tester_cache.TesterCache
}

func (s *TestCaseHarness) RegisterTeardownFunc(teardownFunc func()) {
	s.teardownFuncs = append(s.teardownFuncs, teardownFunc)
}

func (s *TestCaseHarness) RunTeardownFuncs() {
	for _, teardownFunc := range s.teardownFuncs {
		teardownFunc()
	}
}

func (s *TestCaseHarness) NewExecutable() *executable.Executable {
	return s.Executable.Clone()
}
