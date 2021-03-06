package tester_utils

// StageHarness is passed to your Stage's TestFunc.
//
// If the program is a long-lived program that must be alive during the duration of the test (like a Redis server),
// do something like this at the start of your test function:
//
//  if err := stageHarness.Executable.Run(); err != nil {
//     return err
//  }
//  defer stageHarness.Executable.Kill()
//
// If the program is a script that must be executed and then checked for output (like a Git command), use it like this:
//
//  result, err := executable.Run("cat-file", "-p", "sha")
//  if err != nil {
//      return err
//   }
type StageHarness struct {
	// Logger is to be used for all logs generated from the test function.
	Logger *Logger

	// Executable is the program to be tested.
	Executable *Executable
}
