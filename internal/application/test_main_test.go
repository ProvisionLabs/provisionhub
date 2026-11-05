package application

import (
	"os"
	"testing"
)

// Keep legacy default Plugin Manager reads inside disposable application data.
// Individual persistence tests still inject their own t.TempDir databases.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "provisionhub-application-test-data-")
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
