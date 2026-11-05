package cli

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "provisionhub-cli-test-data-")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("PROVISIONHUB_DATA_DIR", directory); err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}
