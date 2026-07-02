package main

import "testing"

func TestAuthenticate(t *testing.T) {
	conf := &Configuration{
		Clients: map[string]Client{
			"token-aaaaaaaaaaaaaaaa": {Name: "alice", Domain: "a.example"},
			"token-bbbbbbbbbbbbbbbb": {Name: "bob", Domain: "b.example"},
		},
	}

	tests := []struct {
		name     string
		auth     string
		wantOK   bool
		wantName string
	}{
		{name: "known token", auth: "token-aaaaaaaaaaaaaaaa", wantOK: true, wantName: "alice"},
		{name: "other known token", auth: "token-bbbbbbbbbbbbbbbb", wantOK: true, wantName: "bob"},
		{name: "unknown token", auth: "nope", wantOK: false},
		{name: "empty token", auth: "", wantOK: false},
		{name: "prefix of a known token", auth: "token-aaaaaaaaaaaaaaa", wantOK: false},
		{name: "known token with trailing byte", auth: "token-aaaaaaaaaaaaaaaax", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, ok := conf.authenticate(tt.auth)
			if ok != tt.wantOK {
				t.Fatalf("authenticate(%q) ok = %v, want %v", tt.auth, ok, tt.wantOK)
			}
			if ok && client.Name != tt.wantName {
				t.Fatalf("authenticate(%q) name = %q, want %q", tt.auth, client.Name, tt.wantName)
			}
		})
	}
}

func TestAuthenticateNoClients(t *testing.T) {
	conf := &Configuration{Clients: map[string]Client{}}
	if _, ok := conf.authenticate("anything"); ok {
		t.Fatal("authenticate against empty client map should not succeed")
	}
}

func TestRedactToken(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: "****"},
		{in: "abcd", want: "****"},
		{in: "abc", want: "****"},
		{in: "abcde", want: "abcd****"},
		{in: "supersecretlongtoken12345", want: "supe****"},
	}

	for _, tt := range tests {
		if got := redactToken(tt.in); got != tt.want {
			t.Errorf("redactToken(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	// The full secret must never appear in the redacted output.
	secret := "supersecretlongtoken12345"
	if got := redactToken(secret); got == secret {
		t.Errorf("redactToken leaked the full token: %q", got)
	}
}
