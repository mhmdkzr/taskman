# NATS Tool

The `nats` tool runs the system `nats` CLI and returns its standard output.
Arguments are passed directly to the process; shell syntax is not interpreted.

## Usage

The tool accepts an `args` array and does not require the `nats` executable name:

```json
{
  "args": [
    "stream",
    "ls"
  ]
}
```

The result is returned as:

```json
{
  "output": "..."
}
```

The system must have `nats` available on `PATH`.
