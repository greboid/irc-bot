package irc

import "testing"

func TestValidFloodProfile(t *testing.T) {
	tests := map[string]bool{
		"restrictive": true,
		"unlimited":   true,
		"":            false,
		"Restrictive": false,
		"none":        false,
	}
	for profile, want := range tests {
		if got := ValidFloodProfile(profile); got != want {
			t.Errorf("ValidFloodProfile(%q) = %v, want %v", profile, got, want)
		}
	}
}
