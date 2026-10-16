package websites

import "testing"

func TestNormalize(t *testing.T) {
	for input, want := range map[string]string{" Go.Dev. ": "go.dev", "LOCALHOST": "localhost", "127.0.0.1": "127.0.0.1", "пример.рф": "xn--e1afmkfd.xn--p1ai"} {
		got, err := Normalize(input)
		if err != nil || got != want {
			t.Fatalf("%q: %q %v", input, got, err)
		}
	}
}
func TestRejectURLsAndMalformedDomains(t *testing.T) {
	for _, s := range []string{"", "https://go.dev", "go.dev/path", "go.dev:443", "*.go.dev", "user@go.dev", "-bad.test", "bad_.test", "bad..test", "192.168.1.1", "go", "go.dev?x=1"} {
		if _, err := Normalize(s); err == nil {
			t.Fatal(s)
		}
	}
}
