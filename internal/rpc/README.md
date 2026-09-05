# JSON-RPC API

Run the application with `-rpc` to enable `POST /rpc`. The endpoint uses JSON-RPC 2.0.

Methods include `system.info`, `models.list`, `models.refresh`, `tools.list`, `agent.list`, `agent.create`, `session.create`, and `session.respond`. Agent turns are the general-purpose interface: the configured agent receives the message and may use the registered file, search, browser, shell, messaging, and todo tools.

Example:

```sh
curl -s http://127.0.0.1:8080/rpc \
  -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"system.info"}'
```
