# pulumi-provider-freebox

Provider Pulumi natif pour Freebox, porté depuis [terraform-provider-freebox](https://github.com/NikolaLohinski/terraform-provider-freebox) **sans** utiliser le Terraform Bridge. Implémentation from scratch avec le [Pulumi Go Provider SDK](https://www.pulumi.com/docs/iac/guides/building-extending/providers/sdks/pulumi-go-provider-sdk/).

## Ressources

- **freebox:index:PortForwarding** – Règle de redirection de port (NAT)
- **freebox:index:VirtualDisk** – Disque virtuel (qcow2/raw) sur la Freebox
- **freebox:index:VirtualMachine** – Machine virtuelle sur la Freebox
- **freebox:index:RemoteFile** – Téléchargement d’un fichier depuis une URL vers la Freebox

## Fonctions (invokes)

- **freebox:index:getApiVersion** – Découverte de l’API Freebox (version, modèle, etc.)
- **freebox:index:getVirtualDisk** – Informations sur un disque virtuel (type, tailles)

## Configuration

Variables d’environnement ou config Pulumi :

| Option       | Env              | Description                          |
|-------------|------------------|--------------------------------------|
| `endpoint`  | `FREEBOX_ENDPOINT`  | URL de la Freebox (défaut: http://mafreebox.freebox.fr) |
| `apiVersion`| `FREEBOX_VERSION`   | Version de l’API (défaut: latest)   |
| `appId`     | `FREEBOX_APP_ID`    | ID d’application API Freebox        |
| `token`     | `FREEBOX_TOKEN`     | Token d’authentification API         |

L’autorisation de l’app se fait via [l’API Freebox](https://dev.freebox.fr/sdk/login/). Vous pouvez utiliser la commande `authorize` du provider Terraform pour obtenir `app_id` et `token`.

## Build

```bash
go build -o bin/pulumi-resource-freebox .
```

Le binaire doit s’appeler `pulumi-resource-freebox` pour être reconnu par Pulumi.

## Utilisation (YAML)

Exemple avec Pulumi YAML en pointant vers le binaire local :

```yaml
name: freebox-example
runtime: yaml
config:
  freebox:endpoint: http://mafreebox.freebox.fr
  freebox:appId: "votre_app_id"
  freebox:token: secret:votre_token

resources:
  pf:
    type: freebox:index:PortForwarding
    properties:
      enabled: true
      ipProtocol: tcp
      portRangeStart: 22
      targetIp: "192.168.1.10"
      comment: "SSH"
```

## Dépendances

- [free-go](https://github.com/NikolaLohinski/free-go) – Client API Freebox
- [pulumi-go-provider](https://github.com/pulumi/pulumi-go-provider) – SDK provider Pulumi en Go

## Licence

Voir [LICENSE](LICENSE).
