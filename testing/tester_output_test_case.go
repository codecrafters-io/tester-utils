package testing

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tester_utils "github.com/codecrafters-io/tester-utils"
	"github.com/codecrafters-io/tester-utils/stdio_mocker"
	"github.com/codecrafters-io/tester-utils/tester_definition"
	"github.com/stretchr/testify/assert"
)

type TesterOutputTestCase struct {
	// CodePath is the path to the code that'll be tested.
	CodePath string

	// ExpectedExitCode is the exit code that we expect the tester to return.
	ExpectedExitCode int

	// UntilStageSlug is the slug of the stage that we want to test until. Either this or StageSlug must be provided.
	UntilStageSlug string

	// SkipAntiCheat is a flag that indicates whether we want to skip the anti-cheat check.
	SkipAntiCheat *bool

	// StageSlugs is the list of stages that we want to test. Either this or UntilStageSlug must be provided.
	StageSlugs []string

	// StdoutFixturePath is the path to the fixture file that contains the expected stdout output.
	StdoutFixturePath string

	// NormalizeOutputFunc is a function that normalizes the tester's output. This is useful for removing things like timestamps.
	NormalizeOutputFunc func([]byte) []byte
}

func buildTestCasesJson(slugs []string) string {
	testCases := []map[string]string{}

	for _, slug := range slugs {
		testCases = append(testCases, map[string]string{
			"slug":              slug,
			"tester_log_prefix": fmt.Sprintf("tester::#%s", strings.ToUpper(slug)),
			"title":             fmt.Sprintf("Stage #%s (%s)", strings.ToUpper(slug), slug),
		})
	}

	testCasesJson, _ := json.Marshal(testCases)
	return string(testCasesJson)
}

func buildTestCasesJsonUntilStageSlug(untilStageSlug string, testerDefinition tester_definition.TesterDefinition) string {
	stageSlugs := []string{}
	foundStageSlug := false

	for _, testCase := range testerDefinition.TestCases {
		stageSlugs = append(stageSlugs, testCase.Slug)

		if testCase.Slug == untilStageSlug {
			foundStageSlug = true
			break
		}
	}

	if !foundStageSlug {
		panic(fmt.Sprintf("Stage slug %s not found", untilStageSlug))
	}

	// Reverse the order of stageSlugs before returning the JSON
	reversedStageSlugs := make([]string, len(stageSlugs))
	copy(reversedStageSlugs, stageSlugs)

	for i, j := 0, len(reversedStageSlugs)-1; i < j; i, j = i+1, j-1 {
		reversedStageSlugs[i], reversedStageSlugs[j] = reversedStageSlugs[j], reversedStageSlugs[i]
	}

	return buildTestCasesJson(reversedStageSlugs)
}

func TestTesterOutput(t *testing.T, testerDefinition tester_definition.TesterDefinition, testCases map[string]TesterOutputTestCase) {
	m := stdio_mocker.NewStdIOMocker()
	defer m.End()

	// Used in testing.IsRecordingOrEvaluatingFixtures()
	_isRecordingOrEvaluatingFixtures = true

	defer func() {
		_isRecordingOrEvaluatingFixtures = false
	}()

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			m.Start()

			skipAntiCheat := true
			if testCase.SkipAntiCheat != nil {
				skipAntiCheat = *testCase.SkipAntiCheat
			}

			if testCase.UntilStageSlug != "" && testCase.StageSlugs != nil && len(testCase.StageSlugs) > 0 {
				t.Fatal("Either UntilStageSlug or StageSlugs must be provided, not both")
			}

			if testCase.UntilStageSlug == "" && len(testCase.StageSlugs) == 0 {
				t.Fatal("Either UntilStageSlug or StageSlugs must be provided")
			}

			var testCasesJson string

			if testCase.UntilStageSlug != "" {
				testCasesJson = buildTestCasesJsonUntilStageSlug(testCase.UntilStageSlug, testerDefinition)
			} else {
				testCasesJson = buildTestCasesJson(testCase.StageSlugs)
			}

			exitCode := runCLIStage(testerDefinition, testCasesJson, testCase.CodePath, skipAntiCheat)
			if !assert.Equal(t, testCase.ExpectedExitCode, exitCode) {
				failWithMockerOutput(t, m)
			}

			m.End()
			CompareOutputWithFixture(t, m.ReadStdout(), testCase.NormalizeOutputFunc, testCase.StdoutFixturePath)
		})
	}
}

func runCLIStage(testerDefinition tester_definition.TesterDefinition, testCasesJson string, relativePath string, skipAntiCheat bool) (exitCode int) {
	// When a command is run with a different working directory, a relative path can cause problems.
	path, err := filepath.Abs(relativePath)
	if err != nil {
		panic(err)
	}

	return tester_utils.RunCLI(map[string]string{
		"CODECRAFTERS_TEST_CASES_JSON": testCasesJson,
		"CODECRAFTERS_REPOSITORY_DIR":  path,
		"CODECRAFTERS_SKIP_ANTI_CHEAT": strconv.FormatBool(skipAntiCheat),
	}, testerDefinition)
}

func failWithMockerOutput(t *testing.T, m *stdio_mocker.IOMocker) {
	m.End()
	t.Errorf("stdout: \n%s\n\nstderr: \n%s", m.ReadStdout(), m.ReadStderr())
	t.FailNow()
}
