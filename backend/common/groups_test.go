package common

import (
	"math/rand"
	"testing"
)

func TestGenerateInviteCode_CorrectLengthAndAlphabet(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		code := generateInviteCode(r)
		if len(code) != inviteCodeLength {
			t.Fatalf("expected length %d, got %d (%q)", inviteCodeLength, len(code), code)
		}
		for _, ch := range code {
			found := false
			for _, allowed := range inviteCodeChars {
				if ch == allowed {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("code %q contains a character outside inviteCodeChars: %q", code, ch)
			}
		}
	}
}
