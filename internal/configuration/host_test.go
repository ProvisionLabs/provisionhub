package configuration

import "testing"

func TestHostFormat(t *testing.T) {
	for input, want := range map[string]string{"localhost": "localhost", "api.example.test": "api.example.test", "API.example.test.": "API.example.test.", "127.0.0.1": "127.0.0.1", "::1": "[::1]", "[2001:db8::1]": "[2001:db8::1]"} {
		value, err := HostLiteral(input)
		if err != nil || value != want {
			t.Fatal(input, value, err)
		}
		if _, err := (Property{Type: "string", Format: "host"}).ValidateValue(input); err != nil {
			t.Fatal(err)
		}
	}
	for _, input := range []string{"", "http://localhost", "localhost:3000", "user@host", "host/path", "a\nb", "a b", "a$VAR", "bad_host", "-host", "host-", "host..test", "999.999.999.999", "[127.0.0.1]", "[::1]:3000", "fe80::1%eth0", "localhost#fragment", "localhost?query"} {
		if _, err := HostLiteral(input); err == nil {
			t.Fatal("accepted", input)
		}
	}
	for _, p := range []Property{{Type: "integer", Format: "host"}, {Type: "string", Format: "script"}, {Type: "string", Format: "host", Default: "host:3000"}} {
		if err := (Schema{Properties: map[string]Property{"host": p}}).Validate(); err == nil {
			t.Fatal(p)
		}
	}
}
