# Patch provider schema before SDK generation and Registry publication.
# Pass the release version: jq --arg version "$VERSION" -f scripts/patch-schema.jq
.version = $version
| .pluginDownloadURL = "github://api.github.com/OlivierPaquien/pulumi-freebox"
| .publisher = "OlivierPaquien"
| .logoUrl = "https://raw.githubusercontent.com/OlivierPaquien/pulumi-freebox/main/docs/logo.svg"
| .language.nodejs.packageName = "pulumi-freebox"
| .language.python.packageName = "pulumi_freebox"
| .language.go.importBasePath = "github.com/OlivierPaquien/pulumi-freebox/sdk/go/freebox"
| .language.go.generateResourceContainerTypes = true
| .language.csharp.rootNamespace = "OlivierPaquien.Pulumi"
