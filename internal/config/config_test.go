package config

import "testing"

func TestAuthConfigValidate(t *testing.T) {
	t.Parallel()

	valid := AuthConfig{
		Enabled:               true,
		Issuer:                "https://id.example",
		ClientID:              "client",
		ClientSecret:          "secret",
		RedirectURL:           "https://app.example/auth/callback",
		PostLogoutRedirectURL: "https://app.example/",
		SessionLifetime:       1,
		SessionIdleTimeout:    1,
	}
	if err := valid.validate(); err != nil {
		t.Fatalf("validate valid config: %v", err)
	}

	valid.Issuer = ""
	if err := valid.validate(); err == nil {
		t.Fatal("validate config with missing issuer succeeded")
	}
}
