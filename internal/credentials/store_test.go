package credentials

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

type fakeVault struct {
	values map[string]string
	err    error
}

func TestNativeVaultIntegration(t *testing.T) {
	if os.Getenv("PROVISIONHUB_TEST_NATIVE_VAULT") != "1" {
		t.Skip("opt-in native vault smoke test")
	}
	store := Native{}
	ref := fmt.Sprintf("smoke-test-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		if err := store.Delete(ref); err != nil {
			t.Error("native vault cleanup failed")
		}
	})
	secret := Secret{AccessToken: "non-sensitive-storage-smoke-test"}
	if err := store.Set(ref, secret); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ref)
	if err != nil || got != secret {
		t.Fatal("native vault round trip failed")
	}
	if err := store.Delete(ref); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ref); !errors.Is(err, ErrMissing) {
		t.Fatal("native vault credential remained after deletion")
	}
}

func (v *fakeVault) Get(service, ref string) (string, error) {
	if v.err != nil {
		return "", v.err
	}
	value, ok := v.values[service+ref]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return value, nil
}
func (v *fakeVault) Set(service, ref, value string) error {
	if v.err != nil {
		return v.err
	}
	v.values[service+ref] = value
	return nil
}
func (v *fakeVault) Delete(service, ref string) error {
	if v.err != nil {
		return v.err
	}
	delete(v.values, service+ref)
	return nil
}
func TestNativeAdapterRoundTripReplaceDeleteAndErrors(t *testing.T) {
	vault := &fakeVault{values: map[string]string{}}
	store := Native{backend: vault}
	if _, err := store.Get("account"); !errors.Is(err, ErrMissing) {
		t.Fatal(err)
	}
	first := Secret{AccessToken: "sensitive-first", RefreshToken: "refresh-secret"}
	second := Secret{AccessToken: "sensitive-replacement"}
	for _, secret := range []Secret{first, second} {
		if err := store.Set("account", secret); err != nil {
			t.Fatal(err)
		}
		got, err := store.Get("account")
		if err != nil || got != secret {
			t.Fatal("round trip failed", err)
		}
	}
	if err := store.Delete("account"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("account"); !errors.Is(err, ErrMissing) {
		t.Fatal(err)
	}
	vault.err = errors.New("backend error includes sensitive-first")
	for _, err := range []error{store.Set("account", first), store.Delete("account")} {
		if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "sensitive-first") {
			t.Fatal("backend error leaked", err)
		}
	}
	if _, err := store.Get("account"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if strings.Contains(fmt.Sprintf("%v %+v %#v", first, first, first), "sensitive-first") {
		t.Fatal("formatter leaked secret")
	}
}
