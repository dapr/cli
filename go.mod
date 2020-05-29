module github.com/dapr/cli

go 1.14

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/Pallinder/sillyname-go v0.0.0-20130730142914-97aeae9e6ba1
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/briandowns/spinner v1.6.1
	github.com/dapr/dapr v0.7.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fatih/color v1.7.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gocarina/gocsv v0.0.0-20190426105157-2fc85fcf0c07
	github.com/hashicorp/go-retryablehttp v0.5.3
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-runewidth v0.0.8 // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/olekukonko/tablewriter v0.0.1
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/phayes/freeport v0.0.0-20171002181615-b8543db493a5
	github.com/shirou/gopsutil v2.20.2+incompatible
	github.com/spf13/cobra v0.0.6
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.4.0
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
)

replace k8s.io/client => github.com/kubernetes-client/go v0.0.0-20190928040339-c757968c4c36
