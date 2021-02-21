package tester_utils

import "fmt"

type Tester struct {
	antiCheatStageRunner stageRunner
	stageRunner          stageRunner
	context              testerContext
}

// NewTester creates a Tester based on the TesterDefinition provided
func NewTester(env map[string]string, definition TesterDefinition) (Tester, error) {
	context, err := getTesterContext(env, definition.ExecutableFileName)
	if err != nil {
		fmt.Printf("%s", err)
		return Tester{}, err
	}

	return Tester{
		context:              context,
		stageRunner:          newStageRunner(definition.Stages),
		antiCheatStageRunner: newQuietStageRunner(definition.AntiCheatStages),
	}, nil
}

func (tester Tester) getQuietExecutable() *Executable {
	return newExecutable(tester.context.executablePath)
}

func (tester Tester) getExecutable() *Executable {
	return newVerboseExecutable(tester.context.executablePath, getLogger(true, "[your_program] ").Plainln)
}

func (tester Tester) PrintDebugContext() {
	if !tester.context.isDebug {
		return
	}

	tester.context.print()
	fmt.Println("")
}

func (tester Tester) RunAntiCheatStages() bool {
	stageRunner := tester.antiCheatStageRunner
	return stageRunner.Run(false, tester.getQuietExecutable())
}

func (tester Tester) RunStages() bool {
	stageRunner := tester.stageRunner.Truncated(tester.context.currentStageSlug)
	return stageRunner.Run(tester.context.isDebug, tester.getExecutable())
}

func (tester Tester) PrintSuccessMessage() {
	fmt.Println("")
	fmt.Println("All tests ran successfully. Congrats!")
	fmt.Println("")
	// TODO: Print next stage!
}
