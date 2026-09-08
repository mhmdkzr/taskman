# `cmd/main`

The `taskman` binary's entrypoint. Deliberately minimal - all CLI logic lives in `internal/cli`:

```go
func main() {
	os.Exit(cli.Run())
}
```

Usage:

```sh
go run ./cmd/main task create --definition "Fix doc drift in balance package"
```
