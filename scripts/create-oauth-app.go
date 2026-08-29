package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/zitadel/zitadel-go/v3/pkg/client"
	applicationv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

func main() {
	// Parse flags
	projectID := flag.String("project-id", "388435721750446087", "Zitadel project ID")
	patToken := flag.String("token", "", "Personal Access Token from Zitadel")
	flag.Parse()

	if *patToken == "" {
		flag.PrintDefaults()
		log.Fatal("token flag is required")
	}

	ctx := context.Background()

	// Create Zitadel instance
	z := zitadel.New(
		"localhost",
		zitadel.WithInsecure("8080"),
	)

	// Create client with PAT authentication
	c, err := client.New(
		ctx,
		z,
		client.WithAuth(client.PAT(*patToken)),
	)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	// Create the OAuth app using Application Service v2
	fmt.Println("Creating OAuth application...")

	resp, err := c.ApplicationServiceV2().CreateApplication(ctx, &applicationv2.CreateApplicationRequest{
		ProjectId: *projectID,
		Name:      "app-bff-local",
		ApplicationType: &applicationv2.CreateApplicationRequest_OidcConfiguration{
			OidcConfiguration: &applicationv2.OIDCConfig{
				ResponseTypes: []applicationv2.OIDCResponseType{
					applicationv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE,
				},
				GrantTypes: []applicationv2.OIDCGrantType{
					applicationv2.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE,
					applicationv2.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN,
				},
				AppType:                applicationv2.OIDCAppType_OIDC_APP_TYPE_WEB,
				AuthMethodType:         applicationv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
				RedirectUris:           []string{"http://localhost:8090/auth/callback"},
				PostLogoutRedirectUris: []string{"http://localhost:8090/"},
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create application: %v", err)
	}

	fmt.Printf("✅ Application created successfully!\n")
	fmt.Printf("Application ID: %s\n", resp.AppId)
	fmt.Printf("Client ID: %s\n", resp.ClientId)
	if resp.ClientSecret != "" {
		fmt.Printf("Client Secret: %s\n", resp.ClientSecret)
		fmt.Printf("\nUpdate .env with:\n")
		fmt.Printf("AUTH_CLIENT_ID=%s\n", resp.ClientId)
		fmt.Printf("AUTH_CLIENT_SECRET=%s\n", resp.ClientSecret)
	}
}
