# Curl Tool

The `curl` tool runs the system `curl` executable and returns its standard
output. Arguments are passed directly to the process; shell syntax is not
interpreted.

## Usage

The tool accepts an `args` array and does not require the `curl` executable name:

```json
{
  "args": [
    "-sS",
    "https://example.com"
  ]
}
```

The result is returned as:

```json
{
  "output": "..."
}
```

The system must have `curl` available on `PATH`.
