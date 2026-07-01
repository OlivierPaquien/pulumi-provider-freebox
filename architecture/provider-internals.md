# Provider internals

Internal architecture of **pulumi-provider-freebox** (the `pulumi-resource-freebox` plugin).

## Layered view

```mermaid
flowchart TB
    subgraph pulumi["Pulumi ecosystem"]
        Engine["Pulumi Engine"]
        Plugin["pulumi-resource-freebox<br/>(Go binary)"]
    end

    subgraph provider["Provider (main.go)"]
        Infer["pulumi-go-provider / infer"]
        Config["Config<br/>endpoint, apiVersion, appId, token"]
        Resources["9 Resources — CRUD"]
        Functions["10 Functions — invoke"]
    end

    subgraph internal["Internal layer"]
        Client["client.go<br/>getFreeboxClient · providerConfig"]
        Utils["freebox_utils.go<br/>polling · checksum · FS tasks"]
        VPNCompat["vpn_compat.go<br/>legacy ↔ modern VPN API"]
        Ops["Domain ops<br/>remote_file_ops · virtual_disk_ops"]
        Log["log.go"]
    end

    subgraph external["External dependencies"]
        FreeGo["free-go<br/>HTTP client + types"]
        FBX["Freebox OS API<br/>/api/{version}/..."]
    end

    Engine --> Plugin
    Plugin --> Infer
    Infer --> Config
    Infer --> Resources
    Infer --> Functions
    Resources --> Client
    Functions --> Client
    Client --> FreeGo
    Resources --> Utils
    Resources --> Ops
    Resources --> VPNCompat
    VPNCompat --> FreeGo
    Ops --> Utils
    Utils --> FreeGo
    FreeGo -->|"HTTPS + X-Fbx-App-Auth"| FBX
```

## Source code layout

```mermaid
flowchart LR
    subgraph entry["Entry point"]
        M[main.go]
    end

    subgraph core["Core"]
        CFG[config.go]
        CL[client.go]
        LG[log.go]
        MD[models.go]
        PL[polling.go]
        FU[freebox_utils.go]
    end

    subgraph resources["Resources (infer.Resource)"]
        PF[resource_port_forwarding.go]
        VD[resource_virtual_disk.go<br/>virtual_disk_ops.go]
        VM[resource_virtual_machine.go<br/>vm_timeouts.go]
        VMP[resource_virtual_machine_power.go]
        DHCP[resource_dhcp_static_lease.go<br/>dhcp_static_lease_helpers.go]
        RF[resource_remote_file.go<br/>remote_file_ops.go]
        VPN_S[resource_vpn_server.go]
        VPN_U[resource_vpn_user.go]
        LAN[resource_lan_config.go]
    end

    subgraph invokes["Functions (infer.Function)"]
        INV[invoke.go<br/>invoke_data.go]
    end

    subgraph compat["Compatibility"]
        VPN[vpn_compat.go]
    end

    M --> CFG
    M --> resources
    M --> invokes
    resources --> CL
    resources --> FU
    resources --> PL
    invokes --> CL
    VPN_S --> VPN
    VPN_U --> VPN
    VPN --> CL
    RF --> FU
    VD --> FU
    VM --> FU
```

## Resources and Freebox API mapping

```mermaid
flowchart TB
    subgraph pulumi_res["Pulumi resources"]
        R1["freebox:fw:PortForwarding"]
        R2["freebox:virtual:Disk"]
        R3["freebox:virtual:Machine"]
        R4["freebox:virtual:MachinePower"]
        R5["freebox:dhcp:StaticLease"]
        R6["freebox:downloads:File"]
        R7["freebox:vpn:Server"]
        R8["freebox:vpn:User"]
        R9["freebox:lan:Config"]
    end

    subgraph freego_api["Freebox API (via free-go)"]
        A1["/fw/redirections/"]
        A2["/vm/disk/ · /vm/"]
        A3["/vm/{id} · WebSocket start/stop"]
        A4["/dhcp/static_lease/"]
        A5["/downloads/ · /fs/"]
        A6["/vpn/openvpn/ or /vpn/openvpn_routed/config/"]
        A7["/vpn/user/ · /vpn/download_config/"]
        A8["/lan/config/"]
    end

    R1 --> A1
    R2 --> A2
    R3 --> A3
    R4 --> A3
    R5 --> A4
    R6 --> A5
    R7 --> A6
    R8 --> A7
    R9 --> A8
```

## Invoke functions (read-only)

| Token | Source | Purpose |
|-------|--------|---------|
| `freebox:api:Version` | `invoke.go` | API discovery (version, model, …) |
| `freebox:virtual:getVirtualDisk` | `invoke_data.go` | Disk metadata |
| `freebox:dhcp:getLease` / `getLeases` | `invoke_data.go` | DHCP static leases |
| `freebox:lan:getConfig` / `getInterfaces` / … | `invoke_data.go` | LAN browser |
| `freebox:virtual:getDistributions` | `invoke.go` | VM OS images |
| `freebox:system:getInfo` | `invoke.go` | System information |

Invokes call `getFreeboxClient` and `free-go` directly — no separate `*_ops.go` layer.

## Cross-cutting patterns

| Pattern | Files | Role |
|---------|-------|------|
| **infer CRUD** | `resource_*.go` | `Create` / `Read` / `Update` / `Delete` + `Args` / `State` types |
| **Shared client** | `client.go` | Merge Pulumi config + env → authenticated `free-go` client |
| **Async polling** | `polling.go`, `freebox_utils.go` | Wait for FS tasks, uploads, downloads, VM disk jobs |
| **VPN compat** | `vpn_compat.go` | Detect legacy VPN API (404 on `/vpn/openvpn/`) → modern paths (`openvpn_routed`, `download_config`); enable server + wait for `started`; recreate user on password change |
| **Domain ops** | `*_ops.go` | Non-trivial logic kept out of CRUD handlers |
| **Logging** | `log.go` | Debug log file (`FREEBOX_DEBUG_LOG`) |

## VPN resource lifecycle (example)

```mermaid
sequenceDiagram
    participant P as Pulumi Engine
    participant I as infer (VpnUser)
    participant C as getFreeboxClient
    participant V as vpn_compat
    participant F as free-go
    participant B as Freebox

    P->>I: Create / Update / Read / Delete
    I->>C: providerConfig(ctx)
    C->>F: Login (appId + token)
    F->>B: POST /login/session

    alt Create
        I->>V: createVPNUserCompat
        V->>F: POST /vpn/user/
        I->>V: getVPNUserClientConfigCompat
        V->>V: ensureOpenVPNServerReady (modern API)
        V->>B: GET download_config/...
    end

    alt Update (modern API)
        I->>V: updateVPNUserCompat
        Note over V: PUT password often returns inval;<br/>fallback: DELETE + POST (recreate)
        I->>V: getVPNUserClientConfigCompat
    end

    I-->>P: State (login, password, ovpnConfig, description*)
```

\* `description` is kept in Pulumi state only; the modern Freebox VPN user API does not persist it.

## Modern vs legacy VPN API

The provider does **not** require `apiVersion: v4`. It uses whatever version you configure (`latest`, `v9`, …).

| Legacy (`free-go` default paths) | Modern (recent Freebox OS) |
|----------------------------------|----------------------------|
| `GET /vpn/openvpn/` | `GET /vpn/openvpn_routed/config/` |
| `GET /vpn/user/{login}/config/openvpn` | `GET /vpn/download_config/{server}/{login}` |
| `PUT /vpn/user/{login}` with login + password | Password-only update often fails → recreate user |

Detection: `GET /vpn/openvpn/` returns `invalid_request` (404) → switch to modern paths in `vpn_compat.go`.

Optional env: `FREEBOX_VPN_SERVER` (default: `openvpn_routed`).

## Testing architecture

```mermaid
flowchart LR
    U["Unit tests<br/>helpers, polling, vpn_compat"]
    M["HTTP mock tests<br/>provider_*_mock_test.go"]
    I["Integration tests<br/>TestProvider + FREEBOX_TOKEN"]

    U --> Code["Provider code"]
    M --> Code
    I -->|"real hardware"| FBX["Freebox"]
```

| Layer | Requires | Coverage |
|-------|----------|----------|
| Unit + mocks | Nothing | ~27% (CI-friendly) |
| Integration | `FREEBOX_TOKEN`, optional `FREEBOX_TEST_*` | ~40–45% |
