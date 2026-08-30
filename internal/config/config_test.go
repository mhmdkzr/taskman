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
		LoginClientPATPath:    "/zitadel/bootstrap/login-client.pat",
		AdminPATPath:          "/zitadel/bootstrap/admin-provisioner.pat",
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
	valid.Issuer = "https://id.example"

	valid.LoginClientPATPath = ""
	if err := valid.validate(); err == nil {
		t.Fatal("validate config with missing login client PAT path succeeded")
	}
	valid.LoginClientPATPath = "/zitadel/bootstrap/login-client.pat"

	valid.AdminPATPath = ""
	if err := valid.validate(); err == nil {
		t.Fatal("validate config with missing admin PAT path succeeded")
	}
}
