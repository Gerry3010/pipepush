package routing

import "testing"

func TestKeyNormalizesEquivalentNames(t *testing.T) {
	want := Key("CI")
	for _, name := range []string{"ci", " CI ", "Ci", "\tci\n"} {
		if got := Key(name); got != want {
			t.Errorf("Key(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestKeyDistinguishesDifferentNames(t *testing.T) {
	if Key("CI") == Key("Deploy") {
		t.Error("different names must produce different keys")
	}
}

func TestKeyEmptyForBlank(t *testing.T) {
	for _, name := range []string{"", "   ", "\t\n"} {
		if got := Key(name); got != "" {
			t.Errorf("Key(%q) = %q, want empty", name, got)
		}
	}
}

func TestKeyLength(t *testing.T) {
	if got := Key("CI"); len(got) != 64 {
		t.Errorf("Key length = %d, want 64 (sha256 hex)", len(got))
	}
}
