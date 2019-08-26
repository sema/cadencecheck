package examples

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/sema/cadencecheck/pkg/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const (
	_goldenTestOutputFilename = "test-output.golden"
	_goldenTestMainFilename   = "main.go"
	_packageTemplate          = "github.com/sema/cadencecheck/examples/%s"
	_goldenFileUpdateFlag     = "UPDATE_GOLDEN"
)

func TestExamplesAndCompareAgainstGoldenOutput(t *testing.T) {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.Name() != _goldenTestMainFilename {
			return nil
		}

		testDir := filepath.Dir(path)
		testPkg := fmt.Sprintf(_packageTemplate, testDir)

		goldenFilePath := filepath.Join(testDir, _goldenTestOutputFilename)

		t.Run(testDir, func(t *testing.T) {
			t.Parallel()

			var outputBuffer bytes.Buffer
			outputWriter := bufio.NewWriter(&outputBuffer)

			err = runner.Run(testPkg, outputWriter, outputWriter, false)
			require.NoError(t, err)

			err = outputWriter.Flush() // force io.Writer to write to the buffer
			require.NoError(t, err)

			actualOutput := normalizeOutput(outputBuffer.Bytes())

			if os.Getenv(_goldenFileUpdateFlag) != "" {
				updateGoldenFile(t, goldenFilePath, actualOutput)
			}

			assertGoldenFile(t, goldenFilePath, actualOutput)
		})

		return nil
	})
	require.NoError(t, err)
}

// normalizeOutput replaces parts of the output to make it stable across different environments (e.g. strips file paths)
func normalizeOutput(actualOutput []byte) []byte {
	r, err := regexp.Compile("[a-zA-Z0-9_\\-/.]+/src/")
	if err != nil {
		log.Fatal(err)
	}

	return r.ReplaceAll(actualOutput, []byte("..snip../src/"))
}

func assertGoldenFile(t *testing.T, goldenPath string, actualOutput []byte) {
	goldenOutput, err := ioutil.ReadFile(goldenPath)
	require.NoError(t, err)
	assert.Equal(t, string(goldenOutput), string(actualOutput))
}

func updateGoldenFile(t *testing.T, goldenPath string, actualOutput []byte) {
	err := ioutil.WriteFile(goldenPath, actualOutput, os.ModePerm)
	require.NoError(t, err)
}
