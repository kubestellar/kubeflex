package init

const (
	URL         = "oci://registry-1.docker.io/bitnamicharts/postgresql:12.5.6"
	RepoName    = "bitnami"
	ChartName   = "postgresql"
	ReleaseName = "postgres"
)

var (
	Args = map[string]string{
		"set": "primary.extendedConfiguration=max_connections=1000,primary.priorityClassName=system-node-critical",
	}
)
