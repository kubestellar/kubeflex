package init

const (
	//URL            = "https://charts.helm.sh/stable"
	URL            = "oci://registry-1.docker.io/bitnamicharts/postgresql:12.5.6"
	RepoName       = "bitnami"
	ChartName      = "postgresql"
	ReleaseName    = "postgres"
	ChartNamespace = "kflex-system"
)

var (
	Args = map[string]string{
		"set": "primary.extendedConfiguration=max_connections=1000",
	}
)
