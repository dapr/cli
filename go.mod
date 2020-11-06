module github.com/dapr/cli

go 1.15

require (
	github.com/Pallinder/sillyname-go v0.0.0-20130730142914-97aeae9e6ba1
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/briandowns/spinner v1.6.1
	github.com/dapr/dapr v0.11.3
	github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/fatih/color v1.7.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gocarina/gocsv v0.0.0-20190426105157-2fc85fcf0c07
	github.com/hashicorp/go-retryablehttp v0.5.3
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-runewidth v0.0.8 // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/olekukonko/tablewriter v0.0.1
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/shirou/gopsutil v2.20.2+incompatible
	github.com/spf13/cobra v0.0.6
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	helm.sh/helm/v3 v3.1.0
	k8s.io/api v0.17.8
	k8s.io/apimachinery v0.17.8
	k8s.io/cli-runtime v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/helm v2.16.10+incompatible
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/structured-merge-diff/v3 v3.0.0-20200116222232-67a7b8c61874 // indirect
)

replace (
	github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2
	k8s.io/client => github.com/kubernetes-client/go v0.0.0-20190928040339-c757968c4c36
)
