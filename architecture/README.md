# Architecture documentation

This folder describes how **pulumi-provider-freebox** fits into a Pulumi deployment and how the provider is structured internally.

## Documents

| Document | Description |
|----------|-------------|
| [global-stack.md](./global-stack.md) | End-to-end flow from `pulumi up` through a Go program (`main.go`) to the Freebox API |
| [provider-internals.md](./provider-internals.md) | Internal layout of the provider plugin (resources, invokes, helpers, VPN compat) |
| [example-program.md](./example-program.md) | Reference stack based on `opaq/bootstrap/pulumi-go` |

## Two different `main.go` binaries

| Binary | Role | Started by |
|--------|------|------------|
| **Your Pulumi program** (e.g. `opaq/bootstrap/pulumi-go/main.go`) | Declares *what* to deploy (VM, DHCP, port forwarding) | Pulumi language host on `pulumi up` |
| **pulumi-provider-freebox** (`pulumi-provider-freebox/main.go`) | Implements *how* each `freebox:*` resource is created/updated/deleted | Pulumi engine (`pulumi-resource-freebox` plugin) |

Install the provider plugin separately:

```bash
pulumi plugin install resource freebox 0.3.1 --file ./bin/pulumi-resource-freebox
```

## External dependencies

- [pulumi-go-provider](https://github.com/pulumi/pulumi-go-provider) (`infer`) — native Go provider SDK
- [free-go](https://github.com/NikolaLohinski/free-go) — HTTP client and types for the Freebox API
- [Freebox OS API](https://dev.freebox.fr/sdk/os/) — REST API on the box (`/api/{version}/…`)
