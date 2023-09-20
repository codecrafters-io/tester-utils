package testing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	tester_utils "github.com/codecrafters-io/tester-utils"
	"github.com/codecrafters-io/tester-utils/executable"
	"github.com/codecrafters-io/tester-utils/stdio_mocker"
	"github.com/stretchr/testify/assert"
)

type TesterOutputTestCase struct {
	// CodePath is the path to the code that'll be tested.
	CodePath            string

	// ExpectedExitCode is the exit code that we expect the tester to return.
	ExpectedExitCode    int

	// StageName is the name of the stage that we want to test.
	StageName           string

	// StdoutFixturePath is the path to the fixture file that contains the expected stdout output.
	StdoutFixturePath   string

	// NormalizeOutputFunc is a function that normalizes the tester's output. This is useful for removing things like timestamps.
	NormalizeOutputFunc func([]byte) []byte
}

func TestTesterOutput(t *testing.T, testerDefinition tester_utils.TesterDefinition, testCases map[string]TesterOutputTestCase) {
	m := stdio_mocker.NewStdIOMocker()
	defer m.End()

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			m.Start()

			exitCode := runCLIStage(testerDefinition, testCase.StageName, testCase.CodePath)
			if !assert.Equal(t, testCase.ExpectedExitCode, exitCode) {
				failWithMockerOutput(t, m)
			}

			m.End()
			compareOutputWithFixture(t, m.ReadStdout(), testCase.NormalizeOutputFunc, testCase.StdoutFixturePath)
		})
	}
}

func compareOutputWithFixture(t *testing.T, testerOutput []byte, normalizeOutputFunc func([]byte) []byte, fixturePath string) {
	shouldRecordFixture := os.Getenv("CODECRAFTERS_RECORD_FIXTURES")

	if shouldRecordFixture == "true" {
		if err := os.MkdirAll(filepath.Dir(fixturePath), os.ModePerm); err != nil {
			panic(err)
		}

		if err := os.WriteFile(fixturePath, testerOutput, 0644); err != nil {
			panic(err)
		}

		return
	}

	fixtureContents, err := os.ReadFile(fixturePath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("Fixture file %s does not exist. To create a new one, use CODECRAFTERS_RECORD_FIXTURES=true", fixturePath)
			return
		}

		panic(err)
	}

	testerOutput = normalizeOutputFunc(testerOutput)
	fixtureContents = normalizeOutputFunc(fixtureContents)

	if bytes.Compare(testerOutput, fixtureContents) != 0 {
		diffExecutablePath, err := exec.LookPath("diff")
		if err != nil {
			panic(err)
		}

		diffExecutable := executable.NewExecutable(diffExecutablePath)

		tmpFile, err := ioutil.TempFile("", "")
		if err != nil {
			panic(err)
		}

		if _, err = tmpFile.Write(testerOutput); err != nil {
			panic(err)
		}

		result, err := diffExecutable.Run("-u", fixturePath, tmpFile.Name())
		if err != nil {
			panic(err)
		}

		// Remove the first two lines of the diff output
		diffContents := bytes.SplitN(result.Stdout, []byte("\n"), 3)[2]

		os.Stdout.Write([]byte("\n\nDifferences detected:\n\n"))
		os.Stdout.Write(diffContents)
		os.Stdout.Write([]byte("\n\nRe-run this test with CODECRAFTERS_RECORD_FIXTURES=true to update fixtures\n\n"))
		t.FailNow()
	}
}

//func normalizeTesterOutput(testerOutput []byte) []byte {
//	re, _ := regexp.Compile("read tcp 127.0.0.1:\\d+->127.0.0.1:6379: read: connection reset by peer")
//	return re.ReplaceAll(testerOutput, []byte("read tcp 127.0.0.1:xxxxx+->127.0.0.1:6379: read: connection reset by peer"))
//}

func runCLIStage(testerDefinition tester_utils.TesterDefinition, slug string, relativePath string) (exitCode int) {
	// When a command is run with a different working directory, a relative path can cause problems.
	path, err := filepath.Abs(relativePath)
	if err != nil {
		panic(err)
	}

	tester, err := tester_utils.NewTester(map[string]string{
		"CODECRAFTERS_CURRENT_STAGE_SLUG": slug,
		"CODECRAFTERS_SUBMISSION_DIR":     path,
	}, testerDefinition)

	if err != nil {
		fmt.Printf("%s", err)
		return 1
	}

	return tester.RunCLI()
}

func failWithMockerOutput(t *testing.T, m *stdio_mocker.IOMocker) {
	m.End()
	t.Error(fmt.Sprintf("stdout: \n%s\n\nstderr: \n%s", m.ReadStdout(), m.ReadStderr()))
	t.FailNow()
}