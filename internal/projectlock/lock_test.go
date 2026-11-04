package projectlock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample() Lock {
	return Lock{LockVersion: 1, Plugins: []Plugin{{ID: "local/nextjs", Version: "1.1.0", Source: "local", ContentSHA256: strings.Repeat("a", 64)}, {ID: "local/nestjs", Version: "1.1.0", Source: "local", ContentSHA256: strings.Repeat("b", 64)}}}
}

func TestSerializationAndValidation(t *testing.T) {
	l := sample()
	data, err := Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(data), "local/nestjs") > strings.Index(string(data), "local/nextjs") {
		t.Fatal("unsorted lock")
	}
	other, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Marshal(other)
	if err != nil || string(again) != string(data) {
		t.Fatalf("unstable serialization: %v", err)
	}
	for _, mutate := range []func(*Lock){func(l *Lock) { l.LockVersion = 2 }, func(l *Lock) { l.Plugins[0].ContentSHA256 = "bad" }, func(l *Lock) { l.Plugins = append(l.Plugins, l.Plugins[0]) }, func(l *Lock) { l.Plugins[0].Source = "official" }} {
		bad := sample()
		mutate(&bad)
		if _, err := Marshal(bad); err == nil {
			t.Fatal("accepted invalid lock")
		}
	}
}

func TestPathAndNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "provisionhub.yaml")
	if err := os.WriteFile(manifest, []byte("manifest"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Save(manifest, sample(), false); err != nil {
		t.Fatal(err)
	}
	if err := Save(manifest, sample(), false); err == nil {
		t.Fatal("overwrote lock")
	}
	if _, found, err := Load(manifest); !found || err != nil {
		t.Fatalf("load: %v %v", found, err)
	}
	if err := Save(manifest, sample(), true); err != nil {
		t.Fatal(err)
	}
	if _, err := Path(filepath.Join(dir, "other.yaml")); err == nil {
		t.Fatal("accepted nonstandard manifest")
	}
}
