package project

import "testing"

func TestNormalizeRemote(t *testing.T) {
	cases := map[string]string{
		"https://github.com/Acme/Foo.git":      "github.com/Acme/Foo",
		"git@github.com:Acme/Foo.git":          "github.com/Acme/Foo",
		"ssh://git@github.com/Acme/Foo":        "github.com/Acme/Foo",
		"https://user:pass@github.com/a/b.git": "github.com/a/b",
	}
	for in, want := range cases {
		got := NormalizeRemote(in)
		if got != want {
			t.Errorf("NormalizeRemote(%q)=%q want %q", in, got, want)
		}
	}
}
