#!/bin/bash
# Direct provisioning of the BFF OAuth app using SQL with eventstore

set -e

echo "Provisioning Zitadel BFF client directly..."

# Get project and instance IDs
INSTANCE_ID=$(docker-compose exec -T postgres psql -U postgres -d zitadel -tc "
  SELECT instance_id FROM projections.projects4 WHERE name = 'ZITADEL' LIMIT 1;
" | tr -d ' \n')

PROJECT_ID=$(docker-compose exec -T postgres psql -U postgres -d zitadel -tc "
  SELECT id FROM projections.projects4 WHERE name = 'ZITADEL' LIMIT 1;
" | tr -d ' \n')

APP_ID="388435721750774599"
CLIENT_ID="388421460865056775"
CLIENT_SECRET="5CeJhXRl9O9oPCIfHB6LaprKRjkX5KcLuvLe9VZ4q3QZWkgL0iK7U9qKHNMO9pmV"

echo "Instance ID: $INSTANCE_ID"
echo "Project ID: $PROJECT_ID"
echo "App ID: $APP_ID"
echo "Client ID: $CLIENT_ID"

# Insert event into eventstore to create the app
# Note: This requires understanding Zitadel's event format, which is complex
# For now, this is a placeholder showing where the fix would go

echo ""
echo "Setup incomplete - requires Zitadel event sourcing knowledge."
echo ""
echo "Alternatively, create the app through the UI:"
echo "1. Open http://localhost:8080/ui/console"
echo "2. Navigate to Projects > ZITADEL"
echo "3. Click 'Create New App'"
echo "4. Use these settings:"
echo "   Name: app-bff-local"
echo "   App Type: Web"
echo "   Auth Method: Basic"
echo "   Grant Types: Authorization Code, Refresh Token"
echo "   Response Type: Code"
echo "   Redirect URIs: http://localhost:8090/auth/callback"
echo "   Post Logout Redirect URIs: http://localhost:8090/"
echo "5. Copy the Client ID and Secret to .env"
echo "6. Restart the app container"
echo ""
echo "Or use the management API with proper authentication:"
echo "See scripts/provision-zitadel-bff.sh for the API approach"
