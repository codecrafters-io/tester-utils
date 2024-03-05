package test_runner

import (
	"fmt"
	"time"

	"github.com/codecrafters-io/tester-utils/logger"
	"github.com/codecrafters-io/tester-utils/test_case_harness"
)

// testRunner is used to run multiple tests
type TestRunnerWorker struct {
	TestRunner TestRunner
	Step       TestRunnerStep
}

func NewTestRunnerWorker(testRunner TestRunner, step TestRunnerStep) *TestRunnerWorker {
	return &TestRunnerWorker{
		TestRunner: testRunner,
		Step:       step,
	}
}

func (w *TestRunnerWorker) Run() bool {
	testCaseHarness := test_case_harness.TestCaseHarness{
		Logger:     w.GetLogger(),
		Executable: w.TestRunner.getExecutable(),
	}

	logger := testCaseHarness.Logger
	logger.Infof("Running tests for %s", w.Step.Title)

	stepResultChannel := make(chan error, 1)
	go func() {
		err := w.Step.TestCase.TestFunc(&testCaseHarness)
		stepResultChannel <- err
	}()

	timeout := w.Step.TestCase.CustomOrDefaultTimeout()

	var err error
	select {
	case stageErr := <-stepResultChannel:
		err = stageErr
	case <-time.After(timeout):
		err = fmt.Errorf("timed out, test exceeded %d seconds", int64(timeout.Seconds()))
	}

	if err != nil {
		logger.Errorf("%s", err)
	} else {
		logger.Successf("Test passed.")
	}

	testCaseHarness.RunTeardownFuncs()

	return err == nil
}

func (w *TestRunnerWorker) GetLogger() *logger.Logger {
	if w.TestRunner.isQuiet {
		return logger.GetQuietLogger("")
	} else {
		return logger.GetLogger(w.TestRunner.isDebug, fmt.Sprintf("[%s] ", w.Step.TesterLogPrefix))
	}
}
