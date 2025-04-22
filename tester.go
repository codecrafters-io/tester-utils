package tester_utils

import (
	"fmt"

	"github.com/codecrafters-io/tester-utils/executable"
	"github.com/codecrafters-io/tester-utils/internal"
	"github.com/codecrafters-io/tester-utils/logger"
	"github.com/codecrafters-io/tester-utils/random"
	"github.com/codecrafters-io/tester-utils/test_runner"
	"github.com/codecrafters-io/tester-utils/tester_context"
	"github.com/codecrafters-io/tester-utils/tester_definition"
)

type Tester struct {
	context    tester_context.TesterContext
	definition tester_definition.TesterDefinition
}

// newTester creates a Tester based on the TesterDefinition provided
func newTester(env map[string]string, definition tester_definition.TesterDefinition) (Tester, error) {
	context, err := tester_context.GetTesterContext(env, definition)
	if err != nil {
		if userError, ok := err.(*internal.UserError); ok {
			return Tester{}, fmt.Errorf("%s", userError.Message)
		}

		return Tester{}, fmt.Errorf("CodeCrafters internal error. Error fetching tester context: %v", err)
	}

	tester := Tester{
		context:    context,
		definition: definition,
	}

	if err := tester.validateContext(); err != nil {
		return Tester{}, fmt.Errorf("CodeCrafters internal error. Error validating tester context: %v", err)
	}

	return tester, nil
}

// RunCLI executes the tester based on user-provided env vars
func RunCLI(env map[string]string, definition tester_definition.TesterDefinition) int {
	random.Init()

	tester, err := newTester(env, definition)
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	tester.printDebugContext()

	// TODO: Validate context here instead of in NewTester?

	if !tester.runStages() {
		return 1
	}

	if !tester.context.ShouldSkipAntiCheatTestCases && !tester.runAntiCheatStages() {
		return 1
	}

	return 0
}

// PrintDebugContext is to be run as early as possible after creating a Tester
func (tester Tester) printDebugContext() {
	if !tester.context.IsDebug {
		return
	}

	tester.context.Print()
	fmt.Println("")
}

// runAntiCheatStages runs any anti-cheat stages specified in the TesterDefinition. Only critical logs are emitted. If
// the stages pass, the user won't see any visible output.
func (tester Tester) runAntiCheatStages() bool {
	return tester.getAntiCheatRunner().Run(false, tester.getQuietExecutable())
}

// runStages runs all the stages upto the current stage the user is attempting. Returns true if all stages pass.
func (tester Tester) runStages() bool {
	return tester.getRunner().Run(tester.context.IsDebug, tester.getExecutable())
}

func (tester Tester) getRunner() test_runner.TestRunner {
	steps := []test_runner.TestRunnerStep{}

	for _, testerContextTestCase := range tester.context.TestCases {
		definitionTestCase := tester.definition.TestCaseBySlug(testerContextTestCase.Slug)

		steps = append(steps, test_runner.TestRunnerStep{
			TestCase:        definitionTestCase,
			TesterLogPrefix: testerContextTestCase.TesterLogPrefix,
			Title:           testerContextTestCase.Title,
		})
	}

	return test_runner.NewTestRunner(steps)
}

func (tester Tester) getAntiCheatRunner() test_runner.TestRunner {
	steps := []test_runner.TestRunnerStep{}

	for index, testCase := range tester.definition.AntiCheatTestCases {
		steps = append(steps, test_runner.TestRunnerStep{
			TestCase:        testCase,
			TesterLogPrefix: fmt.Sprintf("ac-%d", index+1),
			Title:           fmt.Sprintf("AC%d", index+1),
		})
	}

	return test_runner.NewQuietTestRunner(steps) // We only want Critical logs to be emitted for anti-cheat tests
}

func (tester Tester) getQuietExecutable() *executable.Executable {
	return executable.NewExecutable(tester.context.ExecutablePath)
}

func (tester Tester) getExecutable() *executable.Executable {
	return executable.NewVerboseExecutable(tester.context.ExecutablePath, logger.GetLogger(true, "[your_program] ").Plainln)
}

func (tester Tester) validateContext() error {
	for _, testerContextTestCase := range tester.context.TestCases {
		testerDefinitionTestCase := tester.definition.TestCaseBySlug(testerContextTestCase.Slug)

		if testerDefinitionTestCase.Slug != testerContextTestCase.Slug {
			return fmt.Errorf("tester context does not have test case with slug %s", testerContextTestCase.Slug)
		}
	}

	return nil
}
