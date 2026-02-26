package test_runner

import (
	"fmt"
	"time"

	"github.com/codecrafters-io/tester-utils/executable"
	"github.com/codecrafters-io/tester-utils/logger"
	"github.com/codecrafters-io/tester-utils/test_case_harness"
	"github.com/codecrafters-io/tester-utils/tester_definition"
)

type TestRunnerStep struct {
	// TestCase is the test case that'll be run against the user's code.
	TestCase tester_definition.TestCase

	// TesterLogPrefix is the prefix that'll be used for all logs emitted by the tester. Example: "stage-1"
	TesterLogPrefix string

	// Title is the title of the test case. Example: "Stage #1: Bind to a port"
	Title string
}

// testRunner is used to run multiple tests
type TestRunner struct {
	isQuiet bool // Used for anti-cheat tests, where we only want Critical logs to be emitted
	steps   []TestRunnerStep
}

func NewTestRunner(steps []TestRunnerStep) TestRunner {
	return TestRunner{
		steps: steps,
	}
}

func NewQuietTestRunner(steps []TestRunnerStep) TestRunner {
	return TestRunner{isQuiet: true, steps: steps}
}

// Run runs all tests in a stageRunner
func (r TestRunner) Run(isDebug bool, executable *executable.Executable) bool {
	for index, step := range r.steps {
		if index != 0 {
			fmt.Println("")
		}

		testCaseHarness := test_case_harness.TestCaseHarness{
			Logger:     r.getLoggerForStep(isDebug, step),
			Executable: executable.Clone(),
		}

		logger := testCaseHarness.Logger
		logger.Infof("Running tests for %s", step.Title)

		stepResultChannel := make(chan error, 1)
		go func() {
			err := step.TestCase.TestFunc(&testCaseHarness)
			stepResultChannel <- err
		}()

		timeout := step.TestCase.CustomOrDefaultTimeout()

		var err error
		select {
		case stageErr := <-stepResultChannel:
			err = stageErr
		case <-time.After(timeout):
			err = fmt.Errorf("timed out, test exceeded %d seconds", int64(timeout.Seconds()))
		}

		if err != nil {
			r.reportTestError(err, isDebug, logger)
		} else {
			logger.Successf("Test passed.")
		}

		testCaseHarness.RunTeardownFuncs()

		if err != nil {
			return false
		}
	}

	return true
}

func (r TestRunner) getLoggerForStep(isDebug bool, step TestRunnerStep) *logger.Logger {
	if r.isQuiet {
		return logger.GetQuietLogger("")
	} else {
		return logger.GetLogger(isDebug, fmt.Sprintf("[%s] ", step.TesterLogPrefix))
	}
}

func (r TestRunner) reportTestError(err error, isDebug bool, logger *logger.Logger) {
	if err.Error() != "" {
		logger.Errorf("%s", err)
	}
	logger.Errorf("Test failed")
}

// Fuck you, go
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
