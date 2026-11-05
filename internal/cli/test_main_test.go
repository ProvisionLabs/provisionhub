package cli

import (
	"os"
	"path/filepath"
	"testing"
)

var testDatabasePath string

func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "provisionhub-cli-test-data-")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("PROVISIONHUB_DATA_DIR", directory); err != nil {
		panic(err)
	}
	testDatabasePath = filepath.Join(directory, "provisionhub.db")
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}
