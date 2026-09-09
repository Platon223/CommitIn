package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("hash equals plaintext")
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("CheckPassword rejected the correct password")
	}
	if CheckPassword(hash, "wrong password") {
		t.Fatal("CheckPassword accepted a wrong password")
	}
}

func TestGenerateTokenIsRandomAndURLSafe(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		tok, err := GenerateToken()
		if err != nil {
			t.Fatalf("GenerateToken: %v", err)
		}
		if len(tok) != 43 { // 32 bytes, base64 raw-url
			t.Fatalf("unexpected token length %d: %q", len(tok), tok)
		}
		if seen[tok] {
			t.Fatalf("duplicate token generated: %q", tok)
		}
		seen[tok] = true
	}
}

func TestHashTokenIsStable(t *testing.T) {
	const tok = "some-token-value"
	if HashToken(tok) != HashToken(tok) {
		t.Fatal("HashToken is not deterministic")
	}
	if HashToken(tok) == HashToken("other") {
		t.Fatal("HashToken collided for different inputs")
	}
	if len(HashToken(tok)) != 64 { // sha-256 hex
		t.Fatalf("unexpected hash length %d", len(HashToken(tok)))
	}
}
