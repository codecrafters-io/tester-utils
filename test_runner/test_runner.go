package test_runner

import (
	"fmt"

	"github.com/codecrafters-io/tester-utils/executable"
	"github.com/codecrafters-io/tester-utils/logger"
	"github.com/codecrafters-io/tester-utils/tester_context"
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
	TesterContext tester_context.TesterContext
	IsQuiet       bool // Used for anti-cheat tests, where we only want Critical logs to be emitted
	Steps         []TestRunnerStep
}

func NewTestRunnerStepFromTestCase(testerDefinitionTestCase tester_definition.TestCase, testerContextTestCase tester_context.TesterContextTestCase) TestRunnerStep {
	return TestRunnerStep{
		TestCase:        testerDefinitionTestCase,
		TesterLogPrefix: testerContextTestCase.TesterLogPrefix,
		Title:           testerContextTestCase.Title,
	}
}

func NewTestRunner(steps []TestRunnerStep, testerContext tester_context.TesterContext) TestRunner {
	return TestRunner{
		TesterContext: testerContext,
		Steps:         steps,
	}
}

func NewQuietTestRunner(steps []TestRunnerStep, testerContext tester_context.TesterContext) TestRunner {
	return TestRunner{
		TesterContext: testerContext,
		IsQuiet:       true,
		Steps:         steps,
	}
}

// Run runs all tests in a stageRunner
func (r TestRunner) Run() bool {
	workerGroup := new(errgroup.Group)
	workerGroup.SetLimit(8)

	failedStepsChannel := make(chan TestRunnerStep, len(r.Steps))
	passedStepsChannel := make(chan TestRunnerStep, len(r.Steps))

	for _, step := range r.Steps {
		stepCopy := step

		workerGroup.Go(func() error {
			worker := NewTestRunnerWorker(r, stepCopy)
			fmt.Println("Running tests for", stepCopy.Title)
			if err := worker.RunProcess(true); err != nil {
				failedStepsChannel <- stepCopy
			} else {
				passedStepsChannel <- stepCopy
			}

			return nil
		})
	}

	fmt.Println("Waiting for tests to finish...")
	if err := workerGroup.Wait(); err != nil {
		panic(err) // We're only using this for concurrency control
	}

	close(failedStepsChannel)
	close(passedStepsChannel)

	failedSteps := make([]TestRunnerStep, 0, len(r.Steps))

	for step := range failedStepsChannel {
		failedSteps = append(failedSteps, step)
	}

	if len(failedSteps) > 0 {
		fmt.Println("Some tests failed!")
		return false
	}

	passedSteps := make([]TestRunnerStep, 0, len(r.Steps))

	for step := range passedStepsChannel {
		passedSteps = append(passedSteps, step)
	}

	if len(passedSteps) != len(r.Steps) {
		panic("Some steps passed, but not all of them. This should never happen.")
	}

	return true
}

func (r TestRunner) RunStepAsWorker(step TestRunnerStep) (exitCode int) {
	worker := NewTestRunnerWorker(r, step)

	if worker.Run() {
		return 0
	}

	return 1
}

func (r TestRunner) GetStepBySlug(slug string) TestRunnerStep {
	for _, step := range r.Steps {
		if step.TestCase.Slug == slug {
			return step
		}
	}

	panic("No step found with slug: " + slug)
}

func (r TestRunner) getExecutable() *executable.Executable {
	if r.IsQuiet {
		return executable.NewExecutable(r.TesterContext.ExecutablePath)
	} else {
		return executable.NewVerboseExecutable(r.TesterContext.ExecutablePath, logger.GetLogger(true, "[your_program] ").Plainln)
	}
}

func (r TestRunner) reportTestError(err error, logger *logger.Logger) {
	logger.Errorf("%s", err)

	if r.TesterContext.IsDebug {
		logger.Errorf("Test failed")
	} else {
		logger.Errorf("Test failed " +
			"(try setting 'debug: true' in your codecrafters.yml to see more details)")
	}
}
