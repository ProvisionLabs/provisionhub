package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/application"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	info := application.Info()
	want := fmt.Sprintf("%s %s\n", info.Name, info.Version)
	if out.String() != want {
		t.Fatalf("version output = %q, want %q", out.String(), want)
	}
}

func TestHelpDoesNotStartTUI(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "version") || !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("unexpected help output: %q", out.String())
	}
}
