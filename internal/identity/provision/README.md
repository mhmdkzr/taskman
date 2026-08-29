# Identity provisioning

This slice creates and retrieves the application-owned UUIDv7 user identity for a
Zitadel `sub`. The mapping is the only database location that stores the external
identity. Other modules must reference `users.id` and never store `zitadel_sub`.
