# `pkg/migrate`

Small SQLite schema runner used by application startup. It reads `schema.sql`
from an embedded filesystem and applies it to the supplied database.

`Migrate` reads the root `schema.sql`. `MigrateSchema` reads
`<sourceDir>/schema.sql`; SQLite does not support database schemas, so its
`schema` argument is ignored.
