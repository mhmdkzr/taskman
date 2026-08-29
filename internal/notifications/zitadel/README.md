# Zitadel notification webhook

Zitadel's active Email HTTP provider calls `POST /webhooks/zitadel/notifications/<secret>`
directly on the Compose network. The secret is a shared secret in the configured endpoint
URL and is checked on every request. The public Caddy listener has no route for `/webhooks/*`.

Configure the provider endpoint as `http://app:8080/webhooks/zitadel/notifications/<secret>`
on the Compose network. The current Zitadel HTTP provider payload supplies the recipient
and rendered template content; this slice relays it through the configured SMTP provider.
