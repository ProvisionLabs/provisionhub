package configuration

import (
	"reflect"
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func testSchema() Schema {
	return Schema{Properties: map[string]Property{
		"title":    {Type: "string", Default: "hello", MinLength: ptr(1), MaxLength: ptr(20)},
		"enabled":  {Type: "boolean", Default: true},
		"port":     {Type: "integer", Default: 3000, Minimum: ptr(int64(1)), Maximum: ptr(int64(65535))},
		"mode":     {Type: "enum", Options: []string{"small", "large"}, Default: "small"},
		"optional": {Type: "string"},
	}}
}

func TestResolve(t *testing.T) {
	s := testSchema()
	a, err := Resolve(s, Values{"enabled": false, "port": int64(4000)}, "api")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve(s, nil, "billing")
	if err != nil {
		t.Fatal(err)
	}
	if a["enabled"] != false || a["port"] != int64(4000) || b["enabled"] != true || b["port"] != int64(3000) {
		t.Fatal(a, b)
	}
	if _, ok := b["optional"]; ok {
		t.Fatal("optional value synthesized")
	}
	for range 20 {
		next, err := Resolve(s, nil, "api")
		if err != nil || !reflect.DeepEqual(next, b) {
			t.Fatal(next, err)
		}
	}
	a["title"] = "changed"
	if b["title"] != "hello" || s.Properties["title"].Default != "hello" {
		t.Fatal("shared state")
	}
}

func TestInvalidValues(t *testing.T) {
	for _, values := range []Values{{"unknown": true}, {"enabled": "true"}, {"port": 1.5}, {"port": float64(3000)}, {"port": 0}, {"port": 65536}, {"mode": "other"}, {"title": ""}, {"title": nil}} {
		_, err := Resolve(testSchema(), values, "api")
		if err == nil || !strings.Contains(err.Error(), `component "api" configuration.`) {
			t.Fatalf("%v: %v", values, err)
		}
	}
	s := Schema{Properties: map[string]Property{"needed": {Type: "string", Required: true}}}
	if _, err := Resolve(s, nil, "api"); err == nil {
		t.Fatal("missing required accepted")
	}
	for range 20 {
		_, err := Resolve(testSchema(), Values{"z": true, "a": false}, "api")
		if !strings.Contains(err.Error(), "configuration.a:") {
			t.Fatal(err)
		}
	}
}

func TestInvalidSchemas(t *testing.T) {
	for _, p := range []Property{
		{Type: "object"}, {Type: "boolean", Default: "false"}, {Type: "integer", Default: 1.0},
		{Type: "integer", Minimum: ptr(int64(5)), Maximum: ptr(int64(1))},
		{Type: "enum"}, {Type: "enum", Options: []string{"a", "a"}}, {Type: "enum", Options: []string{"a"}, Default: "b"},
		{Type: "boolean", Options: []string{"a"}}, {Type: "string", Minimum: ptr(int64(0))},
		{Type: "string", MinLength: ptr(-1)}, {Type: "string", MinLength: ptr(4), MaxLength: ptr(1)},
	} {
		if err := (Schema{Properties: map[string]Property{"value": p}}).Validate(); err == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
	if err := (Schema{Properties: map[string]Property{"bad.key": {Type: "string"}}}).Validate(); err == nil {
		t.Fatal("bad key")
	}
}

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		typ, raw string
		want     any
	}{{"string", "true", "true"}, {"boolean", "false", false}, {"integer", "42", int64(42)}} {
		value, err := (Property{Type: tc.typ}).Parse(tc.raw)
		if err != nil || value != tc.want {
			t.Fatal(value, err)
		}
	}
	for _, raw := range []string{"yes", "1", "TRUE", ""} {
		if _, err := (Property{Type: "boolean"}).Parse(raw); err == nil {
			t.Fatal(raw)
		}
	}
	for _, raw := range []string{"1.1", "9223372036854775808", "0x12", ""} {
		if _, err := (Property{Type: "integer"}).Parse(raw); err == nil {
			t.Fatal(raw)
		}
	}
}
