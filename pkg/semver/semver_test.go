package semver

import (
	"testing"
)

func TestParseValid(t *testing.T) {
	cases := []struct {
		in string
		ex string
	}{
		{"1.2.3", "1.2.3"},
		{"v1.2.3", "1.2.3"},
		{"1.2.3-alpha", "1.2.3-alpha"},
		{"1.2.3-alpha.1+build.1", "1.2.3-alpha.1+build.1"},
		{"0.0.1", "0.0.1"},
		{"10.20.30-rc.1", "10.20.30-rc.1"},
	}
	for _, c := range cases {
		v, err := Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q) unexpected error: %v", c.in, err)
		}
		if s := v.String(); s != c.ex {
			t.Fatalf("Parse(%q).String() = %q; want %q", c.in, s, c.ex)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []string{"1.2", "a.b.c", "1.2.x", ""}
	for _, c := range cases {
		if _, err := Parse(c); err == nil {
			t.Fatalf("Parse(%q) expected error", c)
		}
	}
}

func TestEquals(t *testing.T) {
	cases := []struct {
		a    string
		b    string
		want bool
	}{
		{"1.2.3+build1", "1.2.3+build2", true},
		{"1.2.3-alpha.1", "1.2.3-alpha.1", true},
		{"1.2.3", "1.2.4", false},
		{"1.2.3-alpha", "1.2.3", false},
	}
	for _, c := range cases {
		a, err := Parse(c.a)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.a, err)
		}
		b, err := Parse(c.b)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.b, err)
		}
		if a.Equals(b) != c.want {
			t.Fatalf("Equals: %q vs %q = %v; want %v", c.a, c.b, a.Equals(b), c.want)
		}
	}
}

func TestGT(t *testing.T) {
	cases := []struct {
		a    string
		b    string
		want bool // a > b
	}{
		{"1.0.0", "0.9.9", true},
		{"1.2.3", "1.2.2", true},
		{"1.2.3", "1.2.3-alpha", true},
		{"1.2.3-alpha", "1.2.3", false},
		{"1.2.3-alpha", "1.2.3-alpha.1", false},
		{"1.2.3-alpha.1", "1.2.3-alpha", true},
		{"1.0.0-alpha", "1.0.0-1", true}, // non-numeric > numeric
		{"1.0.0-2", "1.0.0-1", true},
		{"1.0.0-alpha.2", "1.0.0-alpha.10", false},
	}
	for _, c := range cases {
		a, err := Parse(c.a)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.a, err)
		}
		b, err := Parse(c.b)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.b, err)
		}
		if a.GT(b) != c.want {
			t.Fatalf("GT: %q > %q = %v; want %v", c.a, c.b, a.GT(b), c.want)
		}
	}
}
