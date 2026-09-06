# Store

`store.Store` owns the SQLite connections used by the application. `RW()` is
the read-write handle for migrations, writes, and reads that must immediately
follow a write. `RO()` is opened with SQLite's `mode=ro` URI option and is for
read-only queries; write statements sent through it are rejected by SQLite.

Use `store.Open(path)` for file-backed databases and close the returned store
when the owning process shuts down. In-memory databases are supported by
`OpenDB` for isolated tests, but not by `Open`, because separate SQLite
connections do not share a plain `:memory:` database.
