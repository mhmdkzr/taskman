# Bash Tool

The `bash` tool executes a supplied command through the system `bash` shell.
Shell syntax, pipes, redirects, and environment expansion are supported. The
command is executed with `bash -c`; the `bash` executable must be available on
`PATH`.

## Usage

The tool accepts a `command` string:

```json
{
  "command": "printf 'hello' | tr a-z A-Z"
}
```

The result contains `stdout`, `stderr`, and `exit_code`.
