package test_runner

import (
	"github.com/codecrafters-io/tester-utils/executable"
	"github.com/codecrafters-io/tester-utils/logger"
	"github.com/codecrafters-io/tester-utils/tester_definition"
	"golang.org/x/sync/errgroup"
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
	executablePath string // executablePath is defined in the TesterDefinition and passed in here
	isDebug        bool   // isDebug is fetched from the user's codecrafters.yml file
	isQuiet        bool   // Used for anti-cheat tests, where we only want Critical logs to be emitted
	steps          []TestRunnerStep
}

func NewTestRunner(steps []TestRunnerStep, isDebug bool, executablePath string) TestRunner {
	return TestRunner{
		isDebug:        isDebug,
		executablePath: executablePath,
		steps:          steps,
	}
}

func NewQuietTestRunner(steps []TestRunnerStep, executablePath string) TestRunner {
	return TestRunner{
		isQuiet:        true,
		steps:          steps,
		isDebug:        false,
		executablePath: executablePath,
	}
}

// Run runs all tests in a stageRunner
func (r TestRunner) Run() bool {
	workerGroup := new(errgroup.Group)
	workerGroup.SetLimit(8)

	failedStepsChannel := make(chan TestRunnerStep, len(r.steps))
	passedStepsChannel := make(chan TestRunnerStep, len(r.steps))

	for _, step := range r.steps {
		stepCopy := step

		workerGroup.Go(func() error {
			worker := NewTestRunnerWorker(r, stepCopy)
			if worker.Run() {
				passedStepsChannel <- stepCopy
			} else {
				failedStepsChannel <- stepCopy
			}

			return nil
		})
	}

	if err := workerGroup.Wait(); err != nil {
		panic(err) // We're only using this for concurrency control
	}

	close(failedStepsChannel)
	close(passedStepsChannel)

	failedSteps := make([]TestRunnerStep, 0, len(r.steps))

	for step := range failedStepsChannel {
		failedSteps = append(failedSteps, step)
	}

	if len(failedSteps) > 0 {
		return false
	}

	passedSteps := make([]TestRunnerStep, 0, len(r.steps))

	for step := range passedStepsChannel {
		passedSteps = append(passedSteps, step)
	}

	if len(passedSteps) != len(r.steps) {
		panic("Some steps passed, but not all of them. This should never happen.")
	}

	return true
}

func (r TestRunner) getExecutable() *executable.Executable {
	if r.isQuiet {
		return executable.NewExecutable(r.executablePath)
	} else {
		return executable.NewVerboseExecutable(r.executablePath, logger.GetLogger(true, "[your_program] ").Plainln)
	}
}

func (r TestRunner) reportTestError(err error, logger *logger.Logger) {
	logger.Errorf("%s", err)

	if r.isDebug {
		logger.Errorf("Test failed")
	} else {
		logger.Errorf("Test failed " +
			"(try setting 'debug: true' in your codecrafters.yml to see more details)")
	}
}
