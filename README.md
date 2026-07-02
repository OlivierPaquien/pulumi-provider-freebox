# pulumi-provider-freebox

Native Pulumi provider for Freebox, ported from [terraform-provider-freebox](https://github.com/NikolaLohinski/terraform-provider-freebox) **without** using the Terraform Bridge. Implemented from scratch with the [Pulumi Go Provider SDK](https://www.pulumi.com/docs/iac/guides/building-extending/providers/sdks/pulumi-go-provider-sdk/).

## Resources

| Token | Description |
|-------|-------------|
| `freebox:fw:PortForwarding` | Port forwarding rule (NAT) |
| `freebox:virtual:Disk` | Virtual disk (qcow2/raw) on the Freebox |
| `freebox:virtual:Machine` | Virtual machine configuration and power state |
| `freebox:virtual:MachinePower` | VM power state only (running/stopped), independent of VM config |
| `freebox:dhcp:StaticLease` | DHCP static lease (reserve IPv4 for a MAC address) |
| `freebox:downloads:File` | File on the Freebox (download, copy, upload, extract) |
| `freebox:vpn:Server` | OpenVPN server configuration (singleton per Freebox) |
| `freebox:vpn:User` | OpenVPN user account |
| `freebox:lan:Config` | LAN configuration (singleton per Freebox) |

### Virtual machines: `Machine` vs `MachinePower`

- Use **`freebox:virtual:Machine`** when you manage VM configuration (disk, memory, vCPUs) and optionally its power state via the `status` property (`running` or `stopped`, default `running`).
- Use **`freebox:virtual:MachinePower`** when you want to control power independently from configuration (e.g. start/stop on a schedule without touching VM settings).
- Avoid managing the same VM with both `status` on `Machine` and a `MachinePower` resource — pick one approach per VM.

## Functions (invokes)

| Token | Description |
|-------|-------------|
| `freebox:api:Version` | Freebox API discovery (version, model, etc.) |
| `freebox:virtual:getVirtualDisk` | Virtual disk information (type, sizes) |
| `freebox:dhcp:getLease` | DHCP static lease by MAC address |
| `freebox:dhcp:getLeases` | List all DHCP static leases |
| `freebox:lan:getConfig` | Read LAN configuration |
| `freebox:lan:getInterfaces` | List LAN interfaces |
| `freebox:lan:getInterfaceHosts` | List hosts on a LAN interface |
| `freebox:lan:getInterfaceHost` | Read a single LAN host |
| `freebox:virtual:getDistributions` | VM OS distributions available on the Freebox |
| `freebox:system:getInfo` | Freebox system information |

## Configuration

Environment variables or Pulumi config:

| Option       | Env              | Description                                    |
|-------------|------------------|------------------------------------------------|
| `endpoint`  | `FREEBOX_ENDPOINT`  | Freebox URL (default: http://mafreebox.freebox.fr) |
| `apiVersion`| `FREEBOX_VERSION`   | API version (default: latest)                 |
| `appId`     | `FREEBOX_APP_ID`    | Freebox API application ID                   |
| `token`     | `FREEBOX_TOKEN`     | API authentication token                      |

App authorization is done via the [Freebox API](https://dev.freebox.fr/sdk/login/). Run the provider binary with the `authorize` subcommand to obtain `app_id` and `token`:

```bash
go build -o bin/pulumi-resource-freebox .
./bin/pulumi-resource-freebox authorize
```

Follow the prompts and approve the request on your Freebox. The command prints `FREEBOX_APP_ID` and `FREEBOX_TOKEN` values to use in Pulumi config or environment variables.

**Debug**: Pulumi hides plugin output. The provider also writes to a file so you can follow logs during a `pulumi destroy`:

- **Default**: it first tries `/tmp/pulumi-freebox-provider.log`, then `$HOME/.pulumi/pulumi-freebox-provider.log` if `/tmp` is not writable (e.g. plugin running in a sandbox). On Windows: `%TEMP%\pulumi-freebox-provider.log`.
- **Custom**: set `FREEBOX_DEBUG_LOG=/path/to/file` (if Pulumi passes env to the plugin).

```bash
# In one terminal: destroy
pulumi destroy

# In another: follow logs (or ~/.pulumi/pulumi-freebox-provider.log if /tmp fails)
tail -f /tmp/pulumi-freebox-provider.log
# or
tail -f ~/.pulumi/pulumi-freebox-provider.log
```

**To see logs during `pulumi destroy`**: Pulumi runs the provider from its cache (or from the configured path). After each `go build`, reinstall the plugin so the correct binary is used, and force a known log file with `FREEBOX_DEBUG_LOG` (Pulumi passes env vars to the plugin):

```bash
VERSION="$(gh release view --repo OlivierPaquien/pulumi-provider-freebox --json tagName -q .tagName)"
VERSION="${VERSION#v}"
pulumi plugin install resource freebox "$VERSION" --file ./bin/pulumi-resource-freebox
FREEBOX_DEBUG_LOG=$HOME/freebox-provider.log pulumi destroy
# in another terminal: tail -f $HOME/freebox-provider.log
```

The log file also contains a `[freebox] log file: ...` line at startup to confirm where it is writing.

## Requirements

- Go 1.24+

## Build

```bash
go build -o bin/pulumi-resource-freebox .
```

The binary must be named `pulumi-resource-freebox` to be recognized by Pulumi.

## Tests

### Unit tests

Provider unit tests use a mock HTTP server and do not require a Freebox. Run them with:

```bash
go test -v -run TestProvider_ .
```

They cover invalid endpoint handling and the `getApiVersion` invoke with different configs (env and block).

### Integration tests

Integration tests use the real Freebox API. They are skipped if `FREEBOX_TOKEN` is not set.

| Test | Required env |
|------|----------------|
| PortForwarding | `FREEBOX_TOKEN`, `FREEBOX_APP_ID` |
| RemoteFile | idem (+ optional `FREEBOX_ROOT`) |
| VpnUser | idem — creates a temporary VPN user |
| VpnServer (read) | idem |
| VpnServer (create/delete) | idem + `FREEBOX_TEST_VPN_SERVER=1` (restores prior config after) |
| VirtualMachine | `FREEBOX_TEST_DISK_PATH` (qcow2 on the Freebox) |
| VirtualMachinePower | `FREEBOX_TEST_VM_ID` **or** `FREEBOX_TEST_DISK_PATH` |

- **PortForwarding**: create/delete, create/update/delete + API verification (CheckDestroy).
- **RemoteFile**: download a small file then delete + verify the file no longer exists.
- **VpnUser**: create, update password, delete + verify user is gone. On Freebox OS v4+, the provider enables the OpenVPN server and waits for it to start before downloading `ovpnConfig`.
- **VpnServer**: read config; optional write cycle when `FREEBOX_TEST_VPN_SERVER=1`.
- **VirtualMachine**: create (stopped) then delete.
- **VirtualMachinePower**: ensure VM stopped via provider, read power state, delete (no start/stop cycle).

Optional: `FREEBOX_ROOT` (default `Freebox`) for RemoteFile tests.  
Optional: `FREEBOX_ENDPOINT` if not using `http://mafreebox.freebox.fr`.  
Optional: `FREEBOX_VPN_SERVER` (default `openvpn_routed`) when the Freebox exposes the v4 VPN API (`openvpn_bridge` on some models).

```bash
export FREEBOX_APP_ID=your_app_id
export FREEBOX_TOKEN=your_token
# optional for RemoteFile:
export FREEBOX_ROOT=Freebox
# optional for VM / MachinePower (create a temporary VM):
export FREEBOX_TEST_DISK_PATH=Freebox/VMs/alpine.qcow2
# or reuse an existing VM id:
export FREEBOX_TEST_VM_ID=42
# optional VpnServer write tests (restores config after):
export FREEBOX_TEST_VPN_SERVER=1
go test -v -run '^TestProvider$' .
```

## Usage (YAML)

Example with Pulumi YAML pointing to the local binary:

```yaml
name: freebox-example
runtime: yaml
config:
  freebox:endpoint: http://mafreebox.freebox.fr
  freebox:appId: "your_app_id"
  freebox:token: secret:your_token

resources:
  pf:
    type: freebox:fw:PortForwarding
    properties:
      enabled: true
      ipProtocol: tcp
      portRangeStart: 22
      targetIp: "192.168.1.10"
      comment: "SSH"
```

## Dependencies

- [free-go](https://github.com/NikolaLohinski/free-go) – Freebox API client
- [pulumi-go-provider](https://github.com/pulumi/pulumi-go-provider) – Pulumi Go provider SDK

## License

See [LICENSE](LICENSE).
