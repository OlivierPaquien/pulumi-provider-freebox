# Global stack — from `pulumi up` to Freebox

This diagram shows the full path when you run `pulumi up` on a Go Pulumi program that uses the Freebox provider (example: [opaq/bootstrap/pulumi-go](https://github.com/your-org/opaq/tree/main/bootstrap/pulumi-go)).

## Overview

```mermaid
flowchart TB
    subgraph dev["Developer machine"]
        CMD["pulumi up"]
        PY["Pulumi.yaml<br/>runtime: go"]
        STACK["Pulumi.*.yaml<br/>freebox:endpoint · apiVersion · appId · token<br/>stack:vm_* config"]
        MAIN["main.go<br/>pulumi.Run(...)"]
        SDK["freebox/*.go<br/>RegisterResource(...)"]
    end

    subgraph pulumi_local["Pulumi (local)"]
        CLI["pulumi CLI"]
        ENGINE["Pulumi Engine"]
        LH["Go language host<br/>runs main.go"]
        STATE["State backend<br/>local file / cloud"]
        PLUGIN["Provider plugin<br/>pulumi-resource-freebox"]
    end

    subgraph provider["pulumi-provider-freebox"]
        INFER["pulumi-go-provider / infer"]
        RES["resource_*.go"]
        CLIENT["client.go → free-go"]
        VPN["vpn_compat.go"]
    end

    subgraph box["Freebox"]
        API["Freebox OS API<br/>e.g. /api/v9/..."]
    end

    CMD --> CLI
    PY --> CLI
    STACK --> CLI
    CLI --> ENGINE
    ENGINE --> LH
    LH --> MAIN
    MAIN --> SDK
    SDK -->|"RegisterResource<br/>e.g. freebox:fw:PortForwarding"| ENGINE
    ENGINE <-->|"state + plan"| STATE
    ENGINE <-->|"gRPC provider protocol<br/>Create / Read / Update / Delete"| PLUGIN
    PLUGIN --> INFER
    INFER --> RES
    RES --> CLIENT
    RES --> VPN
    CLIENT --> API
    VPN --> API
```

## Detailed sequence (`pulumi up`)

```mermaid
sequenceDiagram
    autonumber
    participant U as User
    participant CLI as pulumi CLI
    participant ENG as Pulumi Engine
    participant LH as Go language host
    participant M as main.go
    participant FB as freebox/*.go (local SDK)
    participant PR as pulumi-resource-freebox
    participant FG as free-go
    participant BX as Freebox API

    U->>CLI: pulumi up
    CLI->>ENG: load stack + config
    ENG->>LH: run Go program (runtime: go)
    LH->>M: main()

    Note over M: reads stack config<br/>vm_img_url, vm_ip, ...

    M->>FB: NewRemoteFile("ubuntu_image", ...)
    FB->>ENG: RegisterResource("freebox:downloads:File", inputs)
    M->>FB: NewVirtualMachine(..., DependsOn image)
    FB->>ENG: RegisterResource("freebox:virtual:Machine", ...)
    M->>FB: NewDHCPStaticLease(...)
    M->>FB: NewVirtualMachinePower(...)
    M->>FB: NewPortForwarding(...)

    ENG->>ENG: build dependency graph<br/>compute plan (create/update/delete)

    loop For each freebox:* resource
        ENG->>PR: Configure(provider)<br/>endpoint, apiVersion, appId, token
        ENG->>PR: Create / Update / Read / Delete
        PR->>FG: Login + HTTP calls
        FG->>BX: POST/GET/PUT/DELETE /api/v9/...
        BX-->>FG: JSON response
        FG-->>PR: typed result
        PR-->>ENG: resource state (ruleId, vmId, ...)
    end

    ENG->>ENG: persist state
    ENG-->>CLI: summary + outputs
    CLI-->>U: vm_id, vm_ip, ssh_src_port, ...
```

## Program vs provider

```mermaid
flowchart LR
    subgraph program["Pulumi program (your project)"]
        P1["main.go"]
        P2["freebox/port_forwarding.go<br/>RegisterResource(...)"]
    end

    subgraph plugin["Provider plugin (installed separately)"]
        P3["pulumi-provider-freebox/main.go"]
        P4["resource_port_forwarding.go<br/>Create / Read / Update / Delete"]
    end

    P1 --> P2
    P2 -.->|"token freebox:fw:PortForwarding"| P3
    P3 --> P4
```

## Configuration flow

```mermaid
flowchart TB
    YAML["Pulumi stack YAML"]
    YAML -->|"freebox:endpoint<br/>freebox:apiVersion<br/>freebox:appId<br/>freebox:token"| ENG["Pulumi Engine"]
    YAML -->|"stack-specific keys<br/>vm_img_url, vm_ip, ..."| PROG["main.go"]

    ENG -->|"provider config"| PR["pulumi-resource-freebox"]
    PR -->|"providerConfig(ctx)"| CL["getFreeboxClient()"]
    CL -->|"HTTPS + session token"| BX["Freebox"]
```

## Authentication

The provider reads configuration from Pulumi provider config and/or environment variables:

| Pulumi config | Environment variable | Purpose |
|---------------|---------------------|---------|
| `freebox:endpoint` | `FREEBOX_ENDPOINT` | Box URL (default: `http://mafreebox.freebox.fr`) |
| `freebox:apiVersion` | `FREEBOX_VERSION` | API path version (default: `latest`) |
| `freebox:appId` | `FREEBOX_APP_ID` | Application ID |
| `freebox:token` | `FREEBOX_TOKEN` | Private API token |

`free-go` performs the login handshake (`/login` → `/login/session`) and sends `X-Fbx-App-Auth` on subsequent requests.
