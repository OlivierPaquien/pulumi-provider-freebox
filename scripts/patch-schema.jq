# Patch provider schema before SDK generation and Registry publication.
del(.version)
| .pluginDownloadURL = "github://api.github.com/OlivierPaquien/pulumi-freebox"
| .publisher = "OlivierPaquien"
| .language.nodejs.packageName = "pulumi-freebox"
| .language.python.packageName = "pulumi_freebox"
| .language.go.importBasePath = "github.com/OlivierPaquien/pulumi-freebox/sdk/go/freebox"
| .language.go.generateResourceContainerTypes = true
| .language.csharp.rootNamespace = "OlivierPaquien.Pulumi"
