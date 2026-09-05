# Deno Tool

The `deno_cli` tool runs the system `deno` CLI and returns its command output.
Arguments are passed directly to the process; shell syntax is not interpreted.

## Usage

The tool accepts an `args` array and does not require the `deno` executable name:

```json
{
  "args": [
    "--version"
  ]
}
```

The system must have `deno` available on `PATH`.
