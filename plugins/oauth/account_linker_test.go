package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

func TestCreateOrLinkAccount(t *testing.T) {
	t.Parallel()

	t.Run("new user auto created", func(t *testing.T) {
		t.Parallel()

		_, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		now := time.Now()
		profile := &limen.OAuthAccountProfile{
			Provider:             "test",
			ProviderAccountID:    "prov-123",
			Email:                "new@example.com",
			EmailVerified:        true,
			AccessToken:          "at",
			RefreshToken:         "rt",
			AccessTokenExpiresAt: &now,
		}

		result, err := plugin.CreateOrLinkAccount(ctx, profile)
		require.NoError(t, err)
		require.NotNil(t, result.User)
		assert.Equal(t, "new@example.com", result.User.Email)
		assert.NotNil(t, result.User.EmailVerifiedAt)
	})

	t.Run("links to existing user", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "existing@example.com")
		verifyOAuthUser(t, plugin, user)

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-456",
			Email:             "existing@example.com",
			EmailVerified:     true,
			AccessToken:       "at",
		}

		result, err := plugin.CreateOrLinkAccount(ctx, profile)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.User.ID)
	})

	t.Run("updates existing account", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "update@example.com")
		seedOAuthAccount(t, plugin, user.ID, "test", "prov-789")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-789",
			Email:             "update@example.com",
			EmailVerified:     true,
			AccessToken:       "new-at",
			RefreshToken:      "new-rt",
		}

		result, err := plugin.CreateOrLinkAccount(ctx, profile)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.User.ID)
	})

	t.Run("require explicit sign up", func(t *testing.T) {
		t.Parallel()

		_, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption(), WithRequireExplicitSignUp())
		ctx := context.Background()

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-new",
			Email:             "noexist@example.com",
			EmailVerified:     true,
			AccessToken:       "at",
		}

		_, err := plugin.CreateOrLinkAccount(ctx, profile)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("unverified email creates unverified new user", func(t *testing.T) {
		t.Parallel()

		_, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-unverified-new",
			Email:             "unverified-new@example.com",
			EmailVerified:     false,
			AccessToken:       "at",
		}

		result, err := plugin.CreateOrLinkAccount(ctx, profile)
		require.NoError(t, err)
		require.NotNil(t, result.User)
		assert.Nil(t, result.User.EmailVerifiedAt)
	})

	t.Run("unverified email updates existing account", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "existing-unverified@example.com")
		seedOAuthAccount(t, plugin, user.ID, "test", "prov-unverified-existing")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-unverified-existing",
			Email:             "existing-unverified@example.com",
			EmailVerified:     false,
			AccessToken:       "at",
		}

		result, err := plugin.CreateOrLinkAccount(ctx, profile)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.User.ID)
	})

	t.Run("unverified provider email rejected for implicit link", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "implicit-unverified@example.com")
		verifyOAuthUser(t, plugin, user)

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-implicit-unverified",
			Email:             "implicit-unverified@example.com",
			EmailVerified:     false,
			AccessToken:       "at",
		}

		_, err := plugin.CreateOrLinkAccount(ctx, profile)
		assert.ErrorIs(t, err, ErrOAuthEmailNotVerified)
	})

	t.Run("implicit link rejects unverified local email", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		seedOAuthUser(t, l, "local-unverified@example.com")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-local-unverified",
			Email:             "local-unverified@example.com",
			EmailVerified:     true,
			AccessToken:       "at",
		}

		_, err := plugin.CreateOrLinkAccount(ctx, profile)
		assert.ErrorIs(t, err, ErrOAuthLocalEmailNotVerified)
	})
}

func TestLinkAccountToCurrentUser(t *testing.T) {
	t.Parallel()

	t.Run("new link", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "link@example.com")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-link",
			Email:             "link@example.com",
			EmailVerified:     true,
			AccessToken:       "at",
		}

		result, err := plugin.LinkAccountToCurrentUser(ctx, user, profile)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.User.ID)
	})

	t.Run("already linked to same user", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "same@example.com")
		seedOAuthAccount(t, plugin, user.ID, "test", "prov-same")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-same",
			Email:             "same@example.com",
			EmailVerified:     true,
			AccessToken:       "new-at",
		}

		result, err := plugin.LinkAccountToCurrentUser(ctx, user, profile)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.User.ID)
	})

	t.Run("already linked to different user", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		owner := seedOAuthUser(t, l, "owner@example.com")
		seedOAuthAccount(t, plugin, owner.ID, "test", "prov-taken")

		other := seedOAuthUser(t, l, "other@example.com")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-taken",
			Email:             "other@example.com",
			EmailVerified:     true,
			AccessToken:       "at",
		}

		_, err := plugin.LinkAccountToCurrentUser(ctx, other, profile)
		assert.ErrorIs(t, err, ErrAccountAlreadyLinkedToAnotherUser)
	})

	t.Run("different email blocked", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "me@example.com")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-diff-email",
			Email:             "different@example.com",
			EmailVerified:     true,
			AccessToken:       "at",
		}

		_, err := plugin.LinkAccountToCurrentUser(ctx, user, profile)
		assert.ErrorIs(t, err, ErrAccountCannotBeLinkedToDifferentEmail)
	})

	t.Run("different email allowed", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption(), WithAllowLinkingDifferentEmails())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "me@example.com")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-cross",
			Email:             "different@example.com",
			EmailVerified:     true,
			AccessToken:       "at",
		}

		result, err := plugin.LinkAccountToCurrentUser(ctx, user, profile)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.User.ID)
	})

	t.Run("unverified email can be linked by current user", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
		ctx := context.Background()

		user := seedOAuthUser(t, l, "current-unverified@example.com")

		profile := &limen.OAuthAccountProfile{
			Provider:          "test",
			ProviderAccountID: "prov-current-unverified",
			Email:             "current-unverified@example.com",
			EmailVerified:     false,
			AccessToken:       "at",
		}

		result, err := plugin.LinkAccountToCurrentUser(ctx, user, profile)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.User.ID)
	})
}

func TestEncryptDecryptTokens(t *testing.T) {
	t.Parallel()

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()

		_, plugin := newTestOAuthPlugin(t)

		profile := &limen.OAuthAccountProfile{
			AccessToken:  "my-access-token",
			RefreshToken: "my-refresh-token",
			IDToken:      "my-id-token",
		}

		encrypted, err := plugin.encryptTokens(profile)
		require.NoError(t, err)
		assert.NotEqual(t, "my-access-token", encrypted.AccessToken)
		assert.NotEqual(t, "my-refresh-token", encrypted.RefreshToken)
		assert.NotEqual(t, "my-id-token", encrypted.IDToken)

		account := &limen.Account{
			AccessToken:  encrypted.AccessToken,
			RefreshToken: encrypted.RefreshToken,
			IDToken:      encrypted.IDToken,
		}
		decrypted, err := plugin.decryptTokens(account)
		require.NoError(t, err)
		assert.Equal(t, "my-access-token", decrypted.AccessToken)
		assert.Equal(t, "my-refresh-token", decrypted.RefreshToken)
		assert.Equal(t, "my-id-token", decrypted.IDToken)
	})

	t.Run("custom functions", func(t *testing.T) {
		t.Parallel()

		customEncrypt := func(_ []byte, tokens *OAuthTokens) (*OAuthTokens, error) {
			return &OAuthTokens{
				AccessToken:  "custom-" + tokens.AccessToken,
				RefreshToken: "custom-" + tokens.RefreshToken,
				IDToken:      "custom-" + tokens.IDToken,
			}, nil
		}
		customDecrypt := func(_ []byte, tokens *OAuthTokens) (*OAuthTokens, error) {
			return &OAuthTokens{
				AccessToken:  tokens.AccessToken[7:],
				RefreshToken: tokens.RefreshToken[7:],
				IDToken:      tokens.IDToken[7:],
			}, nil
		}

		_, plugin := newTestOAuthPlugin(t, WithEncryptTokens(customEncrypt), WithDecryptTokens(customDecrypt))

		profile := &limen.OAuthAccountProfile{
			AccessToken:  "at",
			RefreshToken: "rt",
			IDToken:      "idt",
		}

		encrypted, err := plugin.encryptTokens(profile)
		require.NoError(t, err)
		assert.Equal(t, "custom-at", encrypted.AccessToken)

		account := &limen.Account{
			AccessToken:  encrypted.AccessToken,
			RefreshToken: encrypted.RefreshToken,
			IDToken:      encrypted.IDToken,
		}
		decrypted, err := plugin.decryptTokens(account)
		require.NoError(t, err)
		assert.Equal(t, "at", decrypted.AccessToken)
		assert.Equal(t, "rt", decrypted.RefreshToken)
	})
}
