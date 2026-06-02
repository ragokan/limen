package oauthgithub

import "testing"

func TestSelectGitHubEmail_VerifiesPreferredEmail(t *testing.T) {
	t.Parallel()

	email, verified := selectGitHubEmail("user@example.com", []githubEmail{
		{Email: "user@example.com", Primary: true, Verified: true},
	})
	if email != "user@example.com" || !verified {
		t.Fatalf("email=%q verified=%v", email, verified)
	}
}

func TestSelectGitHubEmail_UnmatchedPreferredEmailIsUnverified(t *testing.T) {
	t.Parallel()

	email, verified := selectGitHubEmail("public@example.com", []githubEmail{
		{Email: "primary@example.com", Primary: true, Verified: true},
	})
	if email != "public@example.com" || verified {
		t.Fatalf("email=%q verified=%v", email, verified)
	}
}

func TestSelectGitHubEmail_PrefersVerifiedEmailWhenNoProfileEmail(t *testing.T) {
	t.Parallel()

	email, verified := selectGitHubEmail("", []githubEmail{
		{Email: "primary@example.com", Primary: true, Verified: false},
		{Email: "verified@example.com", Verified: true},
	})
	if email != "verified@example.com" || !verified {
		t.Fatalf("email=%q verified=%v", email, verified)
	}
}
