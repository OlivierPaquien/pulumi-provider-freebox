# Patch provider schema before SDK generation and Registry publication.
del(.version)
| .pluginDownloadURL = "github://api.github.com/OlivierPaquien/pulumi-provider-freebox"
| .publisher = "OlivierPaquien"
| .language.nodejs.packageName = "pulumi-freebox"
| .language.python.packageName = "pulumi_freebox"
| .language.go.importBasePath = "github.com/OlivierPaquien/pulumi-provider-freebox/sdk/go/freebox"
| .language.go.generateResourceContainerTypes = true
