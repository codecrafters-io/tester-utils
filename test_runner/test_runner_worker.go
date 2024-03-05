package test_runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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

func (w *TestRunnerWorker) RunProcess(shouldStreamOutput bool) error {
	command := exec.Command(w.TestRunner.TesterContext.TesterExecutablePath)
	command.Env = os.Environ()
	command.Env = append(command.Env, "CODECRAFTERS_IS_WORKER_PROCESS=true")
	command.Env = append(command.Env, fmt.Sprintf("CODECRAFTERS_WORKER_PROCESS_STEP_SLUG=%s", w.Step.TestCase.Slug))

	if shouldStreamOutput {
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
	} else {
		command.Stdout = io.Discard
		command.Stderr = io.Discard
	}

	return command.Run()
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
	if w.TestRunner.IsQuiet {
		return logger.GetQuietLogger("")
	} else {
		return logger.GetLogger(w.TestRunner.TesterContext.IsDebug, fmt.Sprintf("[%s] ", w.Step.TesterLogPrefix))
	}
}
