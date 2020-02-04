module github.com/dapr/cli

go 1.12

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/Pallinder/sillyname-go v0.0.0-20130730142914-97aeae9e6ba1
	github.com/briandowns/spinner v1.6.1
	github.com/dapr/dapr v0.3.0-rc.0.0.20200203194726-e540a7166aea
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fatih/color v1.7.0
	github.com/gocarina/gocsv v0.0.0-20190426105157-2fc85fcf0c07
	github.com/google/uuid v1.1.1
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/olekukonko/tablewriter v0.0.1
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/phayes/freeport v0.0.0-20171002181615-b8543db493a5
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.5.0
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
)

replace k8s.io/client => github.com/kubernetes-client/go v0.0.0-20190928040339-c757968c4c36
