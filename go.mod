module github.com/dapr/cli

go 1.16

require (
	github.com/Pallinder/sillyname-go v0.0.0-20130730142914-97aeae9e6ba1
	github.com/briandowns/spinner v1.6.1
	github.com/dapr/dapr v1.6.0
	github.com/dapr/go-sdk v1.0.0
	github.com/docker/docker v17.12.0-ce-rc1.0.20200916142827-bd33bbf0497b+incompatible
	github.com/emicklei/go-restful v2.10.0+incompatible // indirect
	github.com/fatih/color v1.10.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/gocarina/gocsv v0.0.0-20190426105157-2fc85fcf0c07
	github.com/hashicorp/go-retryablehttp v0.5.3
	github.com/hashicorp/go-version v1.3.0
	github.com/mattn/go-runewidth v0.0.8 // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/olekukonko/tablewriter v0.0.2
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/shirou/gopsutil v3.21.4+incompatible
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.5.3
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/helm v2.16.10+incompatible
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2
	k8s.io/client => github.com/kubernetes-client/go v0.0.0-20190928040339-c757968c4c36
)
