package agentcore

import "testing"

const testHubPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDXvBALRC6pB8/bY8wpyWU5AYk+3NDJxLtJqstVU3OI6 labtether-hub"

// A Hub key payload with an embedded newline must not reach authorized_keys:
// each extra line would be trusted as another key, with its own options.
func TestValidateAuthorizedKeyRejectsExtraLines(t *testing.T) {
	for _, payload := range []string{
		testHubPublicKey + "\nssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDXvBALRC6pB8/bY8wpyWU5AYk+3NDJxLtJqstVU3OI6 extra",
		testHubPublicKey + "\rcommand=\"/bin/sh\" ssh-ed25519 AAAA",
		"not-a-key",
	} {
		if _, err := validateAuthorizedKey(payload); err == nil {
			t.Fatalf("validateAuthorizedKey(%q) accepted an invalid entry", payload)
		}
	}
	got, err := validateAuthorizedKey("  " + testHubPublicKey + "\n")
	if err != nil || got != testHubPublicKey {
		t.Fatalf("validateAuthorizedKey(valid)=%q, %v; want trimmed key", got, err)
	}
}
