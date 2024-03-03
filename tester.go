package tester_utils

import (
	"fmt"

	"github.com/codecrafters-io/tester-utils/random"
	"github.com/codecrafters-io/tester-utils/test_runner"
	"github.com/codecrafters-io/tester-utils/tester_context"
	"github.com/codecrafters-io/tester-utils/tester_definition"
)

type Tester struct {
	context    tester_context.TesterContext
	definition tester_definition.TesterDefinition
}

// NewTester creates a Tester based on the TesterDefinition provided
func NewTester(env map[string]string, definition tester_definition.TesterDefinition) (Tester, error) {
	context, err := tester_context.GetTesterContext(env, definition.ExecutableFileName)
	if err != nil {
		fmt.Println(err.Error())
		return Tester{}, err
	}

	tester := Tester{
		context:    context,
		definition: definition,
	}

	if err := tester.validateContext(); err != nil {
		return Tester{}, err
	}

	return tester, nil
}

// RunCLI executes the tester based on user-provided env vars
func (tester Tester) RunCLI() int {
	random.Init()
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
	return tester.getAntiCheatRunner().Run()
}

// runStages runs all the stages upto the current stage the user is attempting. Returns true if all stages pass.
func (tester Tester) runStages() bool {
	return tester.getRunner().Run()
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

	return test_runner.NewTestRunner(steps, tester.context.IsDebug, tester.context.ExecutablePath)
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

	// We only want Critical logs to be emitted for anti-cheat tests
	return test_runner.NewQuietTestRunner(steps, tester.context.ExecutablePath)
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
