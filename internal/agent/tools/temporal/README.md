# Temporal Tool

The `temporal_cli` tool runs the system `temporal` CLI and returns its standard
output. Arguments are passed directly to the process; shell syntax is not
interpreted.

## Usage

The tool accepts an `args` array and does not require the `temporal` executable
name:

```json
{
  "args": [
    "workflow",
    "list"
  ]
}
```

The system must have `temporal` available on `PATH`.
