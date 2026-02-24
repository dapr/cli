/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	daprRuntime "github.com/dapr/dapr/pkg/runtime"

	cmdruntime "github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/metadata"
	"github.com/dapr/cli/pkg/print"
	runExec "github.com/dapr/cli/pkg/runexec"
	"github.com/dapr/cli/pkg/runfileconfig"
	"github.com/dapr/cli/pkg/standalone"
	daprsyscall "github.com/dapr/cli/pkg/syscall"
	"github.com/dapr/cli/utils"
)

var (
	appPort              int
	profilePort          int
	appID                string
	configFile           string
	port                 int
	grpcPort             int
	internalGRPCPort     int
	maxConcurrency       int
	enableProfiling      bool
	logLevel             string
	protocol             string
	componentsPath       string
	resourcesPaths       []string
	appSSL               bool
	metricsPort          int
	maxRequestBodySize   string
	readBufferSize       string
	unixDomainSocket     string
	enableAppHealth      bool
	appHealthPath        string
	appHealthInterval    int
	appHealthTimeout     int
	appHealthThreshold   int
	enableAPILogging     bool
	apiListenAddresses   string
	schedulerHostAddress string
	runFilePath          string
	appChannelAddress    string
	enableRunK8s         bool
)

const (
	defaultRunTemplateFileName  = "dapr.yaml"
	runtimeWaitTimeoutInSeconds = 60
)

// Flags that are compatible with --run-file
var runFileCompatibleFlags = []string{
	"kubernetes",
	"help",
	"version",
	"runtime-path",
	"log-as-json",
}

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Dapr and (optionally) your application side by side. Supported platforms: Self-hosted",
	Example: `
# Run a .NET application
dapr run --app-id myapp --app-port 5000 -- dotnet run

# Run a Java application
dapr run --app-id myapp -- java -jar myapp.jar

# Run a NodeJs application that listens to port 3000
dapr run --app-id myapp --app-port 3000 -- node myapp.js

# Run a Python application
dapr run --app-id myapp -- python myapp.py

# Run sidecar only
dapr run --app-id myapp

# Run a gRPC application written in Go (listening on port 3000)
dapr run --app-id myapp --app-port 3000 --app-protocol grpc -- go run main.go

# Run a gRPC application written in Go (listening on port 3000) with a different app channel address
dapr run --app-id myapp --app-port 3000 --app-channel-address localhost --app-protocol grpc -- go run main.go


# Run sidecar only specifying dapr runtime installation directory
dapr run --app-id myapp --runtime-path /usr/local/dapr

# Run multiple apps by providing path of a run config file
dapr run --run-file dapr.yaml

# Run multiple apps by providing a directory path containing the run config file(dapr.yaml)
dapr run --run-file /path/to/directory

# Run multiple apps by providing config via stdin
cat dapr.template.yaml | envsubst | dapr run --run-file -

# Run multiple apps in Kubernetes by proficing path of a run config file
dapr run --run-file dapr.yaml -k

# Run multiple apps in Kubernetes by providing a directory path containing the run config file(dapr.yaml)
dapr run --run-file /path/to/directory -k
  `,
	Args: cobra.MinimumNArgs(0),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("placement-host-address", cmd.Flags().Lookup("placement-host-address"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(runFilePath) > 0 {
			// Check for incompatible flags
			incompatibleFlags := detectIncompatibleFlags(cmd)
			if len(incompatibleFlags) > 0 {
				// Print warning message about incompatible flags
				warningMsg := "The following flags are ignored when using --run-file and should be configured in the run file instead: " + strings.Join(incompatibleFlags, ", ")
				print.WarningStatusEvent(os.Stdout, warningMsg)
			}

			runConfigFilePath, err := getRunFilePath(runFilePath)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Failed to get run file path: %v", err)
				os.Exit(1)
			}
			executeRunWithAppsConfigFile(runConfigFilePath, enableRunK8s)
			return
		}
		if len(args) == 0 {
			fmt.Println(print.WhiteBold("WARNING: no application command found."))
		}

		daprDirPath, pathErr := standalone.GetDaprRuntimePath(cmdruntime.GetDaprRuntimePath())
		if pathErr != nil {
			print.FailureStatusEvent(os.Stderr, "Failed to get Dapr install directory: %v", pathErr)
			os.Exit(1)
		}

		// Fallback to default config file if not specified.
		if configFile == "" {
			configFile = standalone.GetDaprConfigPath(daprDirPath)
		}

		// Fallback to default components directory if not specified.
		if componentsPath == "" {
			componentsPath = standalone.GetDaprComponentsPath(daprDirPath)
		}

		if unixDomainSocket != "" {
			// TODO(@daixiang0): add Windows support.
			if runtime.GOOS == string(windowsOsType) {
				print.FailureStatusEvent(os.Stderr, "The unix-domain-socket option is not supported on Windows")
				os.Exit(1)
			} else {
				// use unix domain socket means no port any more.
				print.WarningStatusEvent(os.Stdout, "Unix domain sockets are currently a preview feature")
				port = 0
				grpcPort = 0
			}
		}

		sharedRunConfig := &standalone.SharedRunConfig{
			ConfigFile:         configFile,
			EnableProfiling:    enableProfiling,
			LogLevel:           logLevel,
			MaxConcurrency:     maxConcurrency,
			AppProtocol:        protocol,
			ComponentsPath:     componentsPath,
			ResourcesPaths:     resourcesPaths,
			AppSSL:             appSSL,
			MaxRequestBodySize: maxRequestBodySize,
			HTTPReadBufferSize: readBufferSize,
			EnableAppHealth:    enableAppHealth,
			AppHealthPath:      appHealthPath,
			AppHealthInterval:  appHealthInterval,
			AppHealthTimeout:   appHealthTimeout,
			AppHealthThreshold: appHealthThreshold,
			EnableAPILogging:   enableAPILogging,
			APIListenAddresses: apiListenAddresses,
			DaprdInstallPath:   cmdruntime.GetDaprRuntimePath(),
		}

		// placement-host-address flag handling: only set pointer if flag was explicitly changed
		if cmd.Flags().Changed("placement-host-address") {
			val := viper.GetString("placement-host-address")
			sharedRunConfig.PlacementHostAddr = &val // may be empty => disable
		}

		// scheduler-host-address defaulting/handling
		if cmd.Flags().Changed("scheduler-host-address") {
			val := schedulerHostAddress
			sharedRunConfig.SchedulerHostAddress = &val // may be empty => disable
		} else {
			// Apply version-based defaulting used previously
			addr := validateSchedulerHostAddress(daprVer.RuntimeVersion, schedulerHostAddress)
			if addr != "" {
				sharedRunConfig.SchedulerHostAddress = &addr
			}
		}
		appConfig := &standalone.RunConfig{
			AppID:             appID,
			AppChannelAddress: appChannelAddress,
			AppPort:           appPort,
			HTTPPort:          port,
			GRPCPort:          grpcPort,
			ProfilePort:       profilePort,
			Command:           args,
			MetricsPort:       metricsPort,
			UnixDomainSocket:  unixDomainSocket,
			InternalGRPCPort:  internalGRPCPort,
			SharedRunConfig:   *sharedRunConfig,
		}
		output, err := runExec.NewOutput(appConfig)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
		// TODO: In future release replace following logic with the refactored functions seen below.

		sigCh := make(chan os.Signal, 1)
		daprsyscall.SetupShutdownNotify(sigCh)

		daprRunning := make(chan bool, 1)
		appRunning := make(chan bool, 1)

		go func() {
			var startInfo string
			if unixDomainSocket != "" {
				startInfo = fmt.Sprintf(
					"Starting Dapr with id %s. HTTP Socket: %v. gRPC Socket: %v.",
					output.AppID,
					utils.GetSocket(unixDomainSocket, output.AppID, "http"),
					utils.GetSocket(unixDomainSocket, output.AppID, "grpc"))
			} else {
				startInfo = fmt.Sprintf(
					"Starting Dapr with id %s. HTTP Port: %v. gRPC Port: %v",
					output.AppID,
					output.DaprHTTPPort,
					output.DaprGRPCPort)
			}

			if (daprVer.RuntimeVersion != "edge") && (semver.Compare(fmt.Sprintf("v%v", daprVer.RuntimeVersion), "v1.14.0-rc.1") == -1) {
				print.InfoStatusEvent(os.Stdout, "The scheduler is only compatible with dapr runtime 1.14 onwards.")
				for i, arg := range output.DaprCMD.Args {
					if strings.HasPrefix(arg, "--scheduler-host-address") {
						output.DaprCMD.Args[i] = ""
					}
				}
			}
			print.InfoStatusEvent(os.Stdout, startInfo)

			output.DaprCMD.Stdout = os.Stdout
			output.DaprCMD.Stderr = os.Stderr
			// Set process group so sidecar survives when we exec the app process.
			setDaprProcessGroupForRun(output.DaprCMD)

			err = output.DaprCMD.Start()
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}

			go func() {
				daprdErr := output.DaprCMD.Wait()

				if daprdErr != nil {
					output.DaprErr = daprdErr
					print.FailureStatusEvent(os.Stderr, "The daprd process exited with error code: %s", daprdErr.Error())
				} else {
					print.SuccessStatusEvent(os.Stdout, "Exited Dapr successfully")
				}
				sigCh <- os.Interrupt
			}()

			if appPort <= 0 {
				// If app does not listen to port, we can check for Dapr's sidecar health before starting the app.
				// Otherwise, it creates a deadlock.
				sidecarUp := true

				if unixDomainSocket != "" {
					httpSocket := utils.GetSocket(unixDomainSocket, output.AppID, "http")
					print.InfoStatusEvent(os.Stdout, "Checking if Dapr sidecar is listening on HTTP socket %v", httpSocket)
					err = utils.IsDaprListeningOnSocket(httpSocket, time.Duration(runtimeWaitTimeoutInSeconds)*time.Second)
					if err != nil {
						sidecarUp = false
						print.WarningStatusEvent(os.Stdout, "Dapr sidecar is not listening on HTTP socket: %s", err.Error())
					}

					grpcSocket := utils.GetSocket(unixDomainSocket, output.AppID, "grpc")
					print.InfoStatusEvent(os.Stdout, "Checking if Dapr sidecar is listening on GRPC socket %v", grpcSocket)
					err = utils.IsDaprListeningOnSocket(grpcSocket, time.Duration(runtimeWaitTimeoutInSeconds)*time.Second)
					if err != nil {
						sidecarUp = false
						print.WarningStatusEvent(os.Stdout, "Dapr sidecar is not listening on GRPC socket: %s", err.Error())
					}

				} else {
					print.InfoStatusEvent(os.Stdout, "Checking if Dapr sidecar is listening on HTTP port %v", output.DaprHTTPPort)
					err = utils.IsDaprListeningOnPort(output.DaprHTTPPort, time.Duration(runtimeWaitTimeoutInSeconds)*time.Second)
					if err != nil {
						sidecarUp = false
						print.WarningStatusEvent(os.Stdout, "Dapr sidecar is not listening on HTTP port: %s", err.Error())
					}

					print.InfoStatusEvent(os.Stdout, "Checking if Dapr sidecar is listening on GRPC port %v", output.DaprGRPCPort)
					err = utils.IsDaprListeningOnPort(output.DaprGRPCPort, time.Duration(runtimeWaitTimeoutInSeconds)*time.Second)
					if err != nil {
						sidecarUp = false
						print.WarningStatusEvent(os.Stdout, "Dapr sidecar is not listening on GRPC port: %s", err.Error())
					}
				}

				if sidecarUp {
					print.InfoStatusEvent(os.Stdout, "Dapr sidecar is up and running.")
				} else {
					print.WarningStatusEvent(os.Stdout, "Dapr sidecar might not be responding.")
				}
			}

			daprRunning <- true
		}()

		<-daprRunning

		go func() {
			if output.AppCMD == nil {
				appRunning <- true
				return
			}

			command := args[0]
			var binary string
			binary, err = exec.LookPath(command)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Failed to find command %s: %v", command, err))
				appRunning <- false
				return
			}
			env := output.AppCMD.Env
			if len(env) == 0 {
				env = os.Environ()
			}
			env = append(env, fmt.Sprintf("DAPR_HTTP_PORT=%d", output.DaprHTTPPort))
			env = append(env, fmt.Sprintf("DAPR_GRPC_PORT=%d", output.DaprGRPCPort))

			if startErr := startAppProcessInBackground(output, binary, args, env, sigCh); startErr != nil {
				print.FailureStatusEvent(os.Stderr, startErr.Error())
				appRunning <- false
				return
			}
			appRunning <- true
		}()

		appRunStatus := <-appRunning
		if !appRunStatus {
			// Start App failed, try to stop Dapr and exit.
			err = output.DaprCMD.Process.Kill()
			if err != nil {
				print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Start App failed, try to stop Dapr Error: %s", err))
			} else {
				print.SuccessStatusEvent(os.Stdout, "Start App failed, try to stop Dapr successfully")
			}
			os.Exit(1)
		}

		// Metadata API is only available if app has started listening to port, so wait for app to start before calling metadata API.
		err = metadata.Put(output.DaprHTTPPort, "cliPID", strconv.Itoa(os.Getpid()), output.AppID, unixDomainSocket)
		if err != nil {
			print.WarningStatusEvent(os.Stdout, "Could not update sidecar metadata for cliPID: %s", err.Error())
		}

		if output.AppCMD != nil {
			if output.AppCMD.Process != nil {
				print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Updating metadata for appPID: %d", output.AppCMD.Process.Pid))
				err = metadata.Put(output.DaprHTTPPort, "appPID", strconv.Itoa(output.AppCMD.Process.Pid), output.AppID, unixDomainSocket)
				if err != nil {
					print.WarningStatusEvent(os.Stdout, "Could not update sidecar metadata for appPID: %s", err.Error())
				}
			}

			appCommand := strings.Join(args, " ")
			print.InfoStatusEvent(os.Stdout, "Updating metadata for app command: "+appCommand)
			err = metadata.Put(output.DaprHTTPPort, "appCommand", appCommand, output.AppID, unixDomainSocket)
			if err != nil {
				print.WarningStatusEvent(os.Stdout, "Could not update sidecar metadata for appCommand: %s", err.Error())
			} else {
				print.SuccessStatusEvent(os.Stdout, "You're up and running! Both Dapr and your app logs will appear here.\n")
			}
		} else {
			print.SuccessStatusEvent(os.Stdout, "You're up and running! Dapr logs will appear here.\n")
		}

		<-sigCh
		print.InfoStatusEvent(os.Stdout, "\nterminated signal received: shutting down")

		exitWithError := false

		if output.DaprErr != nil {
			exitWithError = true
			print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error exiting Dapr: %s", output.DaprErr))
		} else if output.DaprCMD.ProcessState == nil || !output.DaprCMD.ProcessState.Exited() {
			err = output.DaprCMD.Process.Kill()
			if err != nil {
				exitWithError = true
				print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error exiting Dapr: %s", err))
			} else {
				print.SuccessStatusEvent(os.Stdout, "Exited Dapr successfully")
			}
		}

		if output.AppErr != nil {
			exitWithError = true
			print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error exiting App: %s", output.AppErr))
		} else if output.AppCMD != nil && output.AppCMD.Process != nil && (output.AppCMD.ProcessState == nil || !output.AppCMD.ProcessState.Exited()) {
			err = output.AppCMD.Process.Kill()
			if err != nil {
				// If the process already exited on its own, treat this as a clean shutdown.
				if errors.Is(err, os.ErrProcessDone) {
					print.SuccessStatusEvent(os.Stdout, "Exited App successfully")
				} else {
					exitWithError = true
					print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error exiting App: %s", err))
				}
			} else {
				print.SuccessStatusEvent(os.Stdout, "Exited App successfully")
			}
		}

		if unixDomainSocket != "" {
			for _, s := range []string{"http", "grpc"} {
				os.Remove(utils.GetSocket(unixDomainSocket, output.AppID, s))
			}
		}

		if exitWithError {
			os.Exit(1)
		}
	},
}

func init() {
	RunCmd.Flags().IntVarP(&appPort, "app-port", "p", -1, "The port your application is listening on")
	RunCmd.Flags().StringVarP(&appID, "app-id", "a", "", "The id for your application, used for service discovery")
	RunCmd.Flags().StringVarP(&configFile, "config", "c", "", "Dapr configuration file")
	RunCmd.Flags().IntVarP(&port, "dapr-http-port", "H", -1, "The HTTP port for Dapr to listen on")
	RunCmd.Flags().IntVarP(&grpcPort, "dapr-grpc-port", "G", -1, "The gRPC port for Dapr to listen on")
	RunCmd.Flags().IntVarP(&internalGRPCPort, "dapr-internal-grpc-port", "I", -1, "The gRPC port for the Dapr internal API to listen on")
	RunCmd.Flags().BoolVar(&enableProfiling, "enable-profiling", false, "Enable pprof profiling via an HTTP endpoint")
	RunCmd.Flags().IntVarP(&profilePort, "profile-port", "", -1, "The port for the profile server to listen on")
	RunCmd.Flags().StringVarP(&logLevel, "log-level", "", "info", "The log verbosity. Valid values are: debug, info, warn, error, fatal, or panic")
	RunCmd.Flags().IntVarP(&maxConcurrency, "app-max-concurrency", "", -1, "The concurrency level of the application, otherwise is unlimited")
	RunCmd.Flags().StringVarP(&protocol, "app-protocol", "P", "http", "The protocol (grpc, grpcs, http, https, h2c) Dapr uses to talk to the application")
	RunCmd.Flags().StringVarP(&componentsPath, "components-path", "d", "", "The path for components directory. Default is $HOME/.dapr/components or %USERPROFILE%\\.dapr\\components")
	RunCmd.Flags().StringSliceVarP(&resourcesPaths, "resources-path", "", []string{}, "The path for resources directory")
	// TODO: Remove below line once the flag is removed in the future releases.
	// By marking this as deprecated, the flag will be hidden from the help menu, but will continue to work. It will show a warning message when used.
	RunCmd.Flags().MarkDeprecated("components-path", "This flag is deprecated and will be removed in the future releases. Use \"resources-path\" flag instead")
	RunCmd.Flags().String("placement-host-address", "localhost", "The address of the placement service. Format is either <hostname> for default port or <hostname>:<port> for custom port. Set to an empty string to disable placement")
	RunCmd.Flags().StringVarP(&schedulerHostAddress, "scheduler-host-address", "", "localhost", "The address of the scheduler service. Format is either <hostname> for default port or <hostname>:<port> for custom port. Set to an empty string to disable scheduler")
	// TODO: Remove below flag once the flag is removed in runtime in future release.
	RunCmd.Flags().BoolVar(&appSSL, "app-ssl", false, "Enable https when Dapr invokes the application")
	RunCmd.Flags().MarkDeprecated("app-ssl", "This flag is deprecated and will be removed in the future releases. Use \"app-protocol\" flag with https or grpcs values instead")
	RunCmd.Flags().IntVarP(&metricsPort, "metrics-port", "M", -1, "The port of metrics on dapr")
	RunCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RunCmd.Flags().StringVarP(&maxRequestBodySize, "max-body-size", "", strconv.Itoa(daprRuntime.DefaultMaxRequestBodySize>>20)+"Mi", "Max size of request body in MB")
	RunCmd.Flags().StringVarP(&readBufferSize, "read-buffer-size", "", strconv.Itoa(daprRuntime.DefaultReadBufferSize>>10)+"Ki", "HTTP header read buffer in KB")
	RunCmd.Flags().StringVarP(&unixDomainSocket, "unix-domain-socket", "u", "", "Path to a unix domain socket dir. If specified, Dapr API servers will use Unix Domain Sockets")
	RunCmd.Flags().BoolVar(&enableAppHealth, "enable-app-health-check", false, "Enable health checks for the application using the protocol defined with app-protocol")
	RunCmd.Flags().StringVar(&appHealthPath, "app-health-check-path", "", "Path used for health checks; HTTP only")
	RunCmd.Flags().IntVar(&appHealthInterval, "app-health-probe-interval", 0, "Interval to probe for the health of the app in seconds")
	RunCmd.Flags().IntVar(&appHealthTimeout, "app-health-probe-timeout", 0, "Timeout for app health probes in milliseconds")
	RunCmd.Flags().IntVar(&appHealthThreshold, "app-health-threshold", 0, "Number of consecutive failures for the app to be considered unhealthy")
	RunCmd.Flags().BoolVar(&enableAPILogging, "enable-api-logging", false, "Log API calls at INFO verbosity. Valid values are: true or false")
	RunCmd.Flags().BoolVarP(&enableRunK8s, "kubernetes", "k", false, "Run the multi-app run template against Kubernetes environment.")
	RunCmd.Flags().StringVar(&apiListenAddresses, "dapr-listen-addresses", "", "Comma separated list of IP addresses that sidecar will listen to")
	RunCmd.Flags().StringVarP(&runFilePath, "run-file", "f", "", "Path to the run template file for the list of apps to run")
	RunCmd.Flags().StringVarP(&appChannelAddress, "app-channel-address", "", utils.DefaultAppChannelAddress, "The network address the application listens on")
	RootCmd.AddCommand(RunCmd)
}

func executeRun(runTemplateName, runFilePath string, apps []runfileconfig.App) (bool, error) {
	var exitWithError bool

	// setup shutdown notify channel.
	sigCh := make(chan os.Signal, 1)
	daprsyscall.SetupShutdownNotify(sigCh)

	runStates := make([]*runExec.RunExec, 0, len(apps))

	// Creates a separate process group ID for current process i.e. "dapr run -f".
	// All the subprocess and their grandchildren inherit this PGID.
	// This is done to provide a better grouping, which can be used to control all the proceses started by "dapr run -f".
	daprsyscall.CreateProcessGroupID()

	for _, app := range apps {
		print.StatusEvent(os.Stdout, print.LogInfo, "Validating config and starting app %q", app.RunConfig.AppID)
		// Set defaults if zero value provided in config yaml.
		app.RunConfig.SetDefaultFromSchema()

		// Adjust scheduler host address defaults for run-file apps (pointer-aware)
		var schedIn string
		if app.RunConfig.SchedulerHostAddress != nil {
			schedIn = *app.RunConfig.SchedulerHostAddress
		}
		schedOut := validateSchedulerHostAddress(daprVer.RuntimeVersion, schedIn)
		if schedOut != "" {
			app.RunConfig.SchedulerHostAddress = &schedOut
		}

		// Validate validates the configs and modifies the ports to free ports, appId etc.
		err := app.RunConfig.Validate()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error validating run config for app %q present in %s: %s", app.RunConfig.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}

		// Get Run Config for different apps.
		runConfig := app.RunConfig
		err = app.CreateDaprdLogFile()
		if err != nil {
			print.StatusEvent(os.Stderr, print.LogFailure, "Error getting daprd log file for app %q present in %s: %s", runConfig.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}

		// Combined multiwriter for logs.
		var appDaprdWriter io.Writer
		// appLogWriter is used when app command is present.
		var appLogWriter io.Writer
		// A custom writer used for trimming ASCII color codes from logs when writing to files.
		var customAppLogWriter io.Writer

		daprdLogWriterCloser := runfileconfig.GetLogWriter(app.DaprdLogWriteCloser, app.DaprdLogDestination)

		if len(runConfig.Command) == 0 {
			print.StatusEvent(os.Stdout, print.LogWarning, "No application command found for app %q present in %s", runConfig.AppID, runFilePath)
			appDaprdWriter = runExec.GetAppDaprdWriter(app, true)
			appLogWriter = app.DaprdLogWriteCloser
		} else {
			err = app.CreateAppLogFile()
			if err != nil {
				print.StatusEvent(os.Stderr, print.LogFailure, "Error getting app log file for app %q present in %s: %s", runConfig.AppID, runFilePath, err.Error())
				exitWithError = true
				break
			}
			appDaprdWriter = runExec.GetAppDaprdWriter(app, false)
			appLogWriter = runfileconfig.GetLogWriter(app.AppLogWriteCloser, app.AppLogDestination)
		}
		customAppLogWriter = print.CustomLogWriter{W: appLogWriter}
		runState, err := startDaprdAndAppProcesses(&runConfig, app.AppDirPath, sigCh,
			daprdLogWriterCloser, daprdLogWriterCloser, customAppLogWriter, customAppLogWriter)
		if err != nil {
			print.StatusEvent(appDaprdWriter, print.LogFailure, "Error starting Dapr and app (%q): %s", app.AppID, err.Error())
			exitWithError = true
			break
		}
		// Store runState in an array.
		runStates = append(runStates, runState)

		// Metadata API is only available if app has started listening to port, so wait for app to start before calling metadata API.
		putCLIProcessIDInMeta(runState, os.Getpid())

		// Update extended metadata with run file path.
		putRunFilePathInMeta(runState, runFilePath)

		// Update extended metadata with run file path.
		putRunTemplateNameInMeta(runState, runTemplateName)

		// Update extended metadata with app log file path.
		if app.AppLogDestination != standalone.Console {
			putAppLogFilePathInMeta(runState, app.AppLogFileName)
		}

		// Update extended metadata with daprd log file path.
		if app.DaprdLogDestination != standalone.Console {
			putDaprLogFilePathInMeta(runState, app.DaprdLogFileName)
		}

		if runState.AppCMD.Command != nil {
			putAppCommandInMeta(runConfig, runState)

			if runState.AppCMD.Command.Process != nil {
				putAppProcessIDInMeta(runState)
				// Attach a windows job object to the app process.
				utils.AttachJobObjectToProcess(strconv.Itoa(os.Getpid()), runState.AppCMD.Command.Process)
			}
		}

		print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogSuccess, "You're up and running! Dapr logs will appear here.\n")
		logInformationalStatusToStdout(app)
	}
	// If all apps have been started and there are no errors in starting the apps wait for signal from sigCh.
	if !exitWithError {
		// After all apps started wait for sigCh.
		<-sigCh
		// To add a new line in Stdout.
		fmt.Println()
		print.InfoStatusEvent(os.Stdout, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")
	}

	// Stop daprd and app processes for each runState.
	closeError := gracefullyShutdownAppsAndCloseResources(runStates, apps)

	for _, app := range apps {
		runConfig := app.RunConfig
		if runConfig.UnixDomainSocket != "" {
			for _, s := range []string{"http", "grpc"} {
				os.Remove(utils.GetSocket(runConfig.UnixDomainSocket, runConfig.AppID, s))
			}
		}
	}

	return exitWithError, closeError
}

func logInformationalStatusToStdout(app runfileconfig.App) {
	print.InfoStatusEvent(os.Stdout, "Started Dapr with app id %q. HTTP Port: %d. gRPC Port: %d",
		app.AppID, app.RunConfig.HTTPPort, app.RunConfig.GRPCPort)
	print.InfoStatusEvent(os.Stdout, "Writing log files to directory : %s", app.GetLogsDir())
}

func gracefullyShutdownAppsAndCloseResources(runState []*runExec.RunExec, apps []runfileconfig.App) error {
	for _, s := range runState {
		stopDaprdAndAppProcesses(s)
	}
	var err error
	// close log file resources.
	for _, app := range apps {
		hasErr := app.CloseAppLogFile()
		if err == nil && hasErr != nil {
			err = hasErr
		}
		hasErr = app.CloseDaprdLogFile()
		if err == nil && hasErr != nil {
			err = hasErr
		}
	}
	return err
}

func executeRunWithAppsConfigFile(runFilePath string, k8sEnabled bool) {
	config, apps, err := getRunConfigFromRunFile(runFilePath)
	if err != nil {
		print.StatusEvent(os.Stdout, print.LogFailure, "Error getting apps from config file: %s", err)
		os.Exit(1)
	}
	if len(apps) == 0 {
		print.StatusEvent(os.Stdout, print.LogFailure, "No apps to run")
		os.Exit(1)
	}
	var exitWithError bool
	var closeErr error
	if !k8sEnabled {
		exitWithError, closeErr = executeRun(config.Name, runFilePath, apps)
	} else {
		exitWithError, closeErr = kubernetes.Run(runFilePath, config)
	}
	if exitWithError {
		if closeErr != nil {
			print.StatusEvent(os.Stdout, print.LogFailure, "Error closing resources: %s", closeErr)
		}
		os.Exit(1)
	}
}

// populate the scheduler host address based on the dapr version.
func validateSchedulerHostAddress(version, address string) string {
	// If no SchedulerHostAddress is supplied, set it to default value.
	if semver.Compare(fmt.Sprintf("v%v", version), "v1.15.0-rc.0") == 1 {
		if address == "" {
			return "localhost"
		}
	}
	return address
}

func getRunConfigFromRunFile(runFilePath string) (runfileconfig.RunFileConfig, []runfileconfig.App, error) {
	config := runfileconfig.RunFileConfig{}
	apps, err := config.GetApps(runFilePath)
	return config, apps, err
}

// startDaprdAndAppProcesses is a function to start the App process and the associated Daprd process.
// This should be called as a blocking function call.
func startDaprdAndAppProcesses(runConfig *standalone.RunConfig, commandDir string, sigCh chan os.Signal,
	daprdOutputWriter io.Writer, daprdErrorWriter io.Writer,
	appOutputWriter io.Writer, appErrorWriter io.Writer,
) (*runExec.RunExec, error) {
	daprRunning := make(chan bool, 1)
	appRunning := make(chan bool, 1)

	daprCMD, err := runExec.GetDaprCmdProcess(runConfig)
	if err != nil {
		print.StatusEvent(daprdErrorWriter, print.LogFailure, "Error getting daprd command with args : %s", err.Error())
		return nil, err
	}
	if strings.TrimSpace(commandDir) != "" {
		daprCMD.Command.Dir = commandDir
	}
	daprCMD.WithOutputWriter(daprdOutputWriter)
	daprCMD.WithErrorWriter(daprdErrorWriter)
	daprCMD.SetStdout()
	daprCMD.SetStderr()

	appCmd, err := runExec.GetAppCmdProcess(runConfig)
	if err != nil {
		print.StatusEvent(appErrorWriter, print.LogFailure, "Error getting app command with args : %s", err.Error())
		return nil, err
	}
	if appCmd.Command != nil {
		// If an app is being run, set the command directory for that app.
		// appCmd does not need to call SetStdout and SetStderr since output is being read processed and then written
		// to the output and error writers for an app.
		appCmd.WithOutputWriter(appOutputWriter)
		appCmd.WithErrorWriter(appErrorWriter)
		if strings.TrimSpace(commandDir) != "" {
			appCmd.Command.Dir = commandDir
		}
	}

	runState := runExec.New(runConfig, daprCMD, appCmd)

	startErrChan := make(chan error, 1)

	// Start daprd process.
	go startDaprdProcess(runConfig, runState, daprRunning, sigCh, startErrChan)

	// Wait for daprRunning channel output.
	if daprStarted := <-daprRunning; !daprStarted {
		startErr := <-startErrChan
		print.StatusEvent(daprdErrorWriter, print.LogFailure, "Error starting daprd process: %s", startErr.Error())
		return nil, startErr
	}

	// No application command is present.
	if appCmd.Command == nil {
		print.StatusEvent(appOutputWriter, print.LogWarning, "No application command present")
		return runState, nil
	}

	if strings.TrimSpace(runConfig.Command[0]) == "" {
		noCmdErr := errors.New("exec: no command")
		print.StatusEvent(appErrorWriter, print.LogFailure, "Error starting app process: %s", noCmdErr.Error())
		_ = killDaprdProcess(runState)
		return nil, noCmdErr
	}

	// Start App process.
	go startAppProcess(runConfig, runState, appRunning, sigCh, startErrChan)

	// Wait for appRunnning channel output.
	if appStarted := <-appRunning; !appStarted {
		startErr := <-startErrChan
		print.StatusEvent(appErrorWriter, print.LogFailure, "Error starting app process: %s", startErr.Error())
		// Start App failed so try to stop daprd process.
		err = killDaprdProcess(runState)
		if err != nil {
			print.StatusEvent(daprdErrorWriter, print.LogFailure, "Error stopping daprd process: %s", err.Error())
			print.StatusEvent(appErrorWriter, print.LogFailure, "Error stopping daprd process: %s", err.Error())
		}
		// Return the error from starting the app process.
		return nil, startErr
	}
	return runState, nil
}

// stopDaprdAndAppProcesses is a function to stop the App process and the associated Daprd process
// This should be called as a blocking function call.
func stopDaprdAndAppProcesses(runState *runExec.RunExec) bool {
	var err error
	print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogInfo, "\ntermination signal received: shutting down")
	// Only if app command is present and
	// if two different output writers are present run the following print statement.
	if runState.AppCMD.Command != nil && runState.AppCMD.OutputWriter != runState.DaprCMD.OutputWriter {
		print.StatusEvent(runState.AppCMD.OutputWriter, print.LogInfo, "\ntermination signal received: shutting down")
	}

	exitWithError := false

	daprErr := runState.DaprCMD.CommandErr

	if daprErr != nil {
		exitWithError = true
		print.StatusEvent(runState.DaprCMD.ErrorWriter, print.LogFailure, "Error exiting Dapr: %s", daprErr)
	} else if runState.DaprCMD.Command.ProcessState == nil || !runState.DaprCMD.Command.ProcessState.Exited() {
		err = killDaprdProcess(runState)
		if err != nil {
			exitWithError = true
		}
	}
	appErr := runState.AppCMD.CommandErr

	if appErr != nil {
		exitWithError = true
		print.StatusEvent(runState.AppCMD.ErrorWriter, print.LogFailure, "Error exiting App: %s", appErr)
	} else if runState.AppCMD.Command != nil && runState.AppCMD.Command.Process != nil && (runState.AppCMD.Command.ProcessState == nil || !runState.AppCMD.Command.ProcessState.Exited()) {
		err = killAppProcess(runState)
		if err != nil {
			exitWithError = true
		}
	}
	return exitWithError
}

// startAppsProcess, starts the App process and calls wait in a goroutine
// This function should be called as a goroutine.
func startAppProcess(runConfig *standalone.RunConfig, runE *runExec.RunExec,
	appRunning chan bool, sigCh chan os.Signal, errorChan chan error,
) {
	if runE.AppCMD.Command == nil {
		appRunning <- true
		return
	}

	stdErrPipe, pipeErr := runE.AppCMD.Command.StderrPipe()
	if pipeErr != nil {
		print.StatusEvent(runE.AppCMD.ErrorWriter, print.LogFailure, "Error creating stderr for App %q : %s", runE.AppID, pipeErr.Error())
		errorChan <- pipeErr
		appRunning <- false
		return
	}

	stdOutPipe, pipeErr := runE.AppCMD.Command.StdoutPipe()
	if pipeErr != nil {
		print.StatusEvent(runE.AppCMD.ErrorWriter, print.LogFailure, "Error creating stdout for App %q : %s", runE.AppID, pipeErr.Error())
		errorChan <- pipeErr
		appRunning <- false
		return
	}

	errScanner := bufio.NewScanner(stdErrPipe)
	outScanner := bufio.NewScanner(stdOutPipe)
	go func() {
		for errScanner.Scan() {
			fmt.Fprintln(runE.AppCMD.ErrorWriter, print.Blue(fmt.Sprintf("== APP - %s == %s", runE.AppID,
				errScanner.Text())))
		}
	}()

	go func() {
		for outScanner.Scan() {
			fmt.Fprintln(runE.AppCMD.OutputWriter, print.Blue(fmt.Sprintf("== APP - %s == %s", runE.AppID, outScanner.Text())))
		}
	}()

	err := runE.AppCMD.Command.Start()
	if err != nil {
		print.StatusEvent(runE.AppCMD.ErrorWriter, print.LogFailure, err.Error())
		errorChan <- err
		appRunning <- false
		return
	}

	go func() {
		appErr := runE.AppCMD.Command.Wait()

		if appErr != nil {
			runE.AppCMD.CommandErr = appErr
			print.StatusEvent(runE.AppCMD.ErrorWriter, print.LogFailure, "The App process exited with error code: %s", appErr.Error())
		} else {
			print.StatusEvent(runE.AppCMD.OutputWriter, print.LogSuccess, "Exited App successfully")
		}
	}()

	appRunning <- true
}

// startDaprdProcess, starts the Daprd process and calls wait in a goroutine
// This function should be called as a goroutine.
func startDaprdProcess(runConfig *standalone.RunConfig, runE *runExec.RunExec,
	daprRunning chan bool, sigCh chan os.Signal, errorChan chan error,
) {
	var startInfo string
	if runConfig.UnixDomainSocket != "" {
		startInfo = fmt.Sprintf(
			"Starting Dapr with id %s. HTTP Socket: %v. gRPC Socket: %v.",
			runE.AppID,
			utils.GetSocket(unixDomainSocket, runE.AppID, "http"),
			utils.GetSocket(unixDomainSocket, runE.AppID, "grpc"))
	} else {
		startInfo = fmt.Sprintf(
			"Starting Dapr with id %s. HTTP Port: %v. gRPC Port: %v",
			runE.AppID,
			runE.DaprHTTPPort,
			runE.DaprGRPCPort)
	}
	print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, startInfo)

	// If DAPR_HOST_IP is not set, set to localhost.
	if _, ok := os.LookupEnv("DAPR_HOST_IP"); !ok {
		runE.DaprCMD.Command.Env = append(runE.DaprCMD.Command.Environ(), "DAPR_HOST_IP=127.0.0.1")
	}

	err := runE.DaprCMD.Command.Start()
	if err != nil {
		errorChan <- err
		daprRunning <- false
		return
	}
	go func() {
		daprdErr := runE.DaprCMD.Command.Wait()
		if daprdErr != nil {
			runE.DaprCMD.CommandErr = daprdErr
			print.StatusEvent(runE.DaprCMD.ErrorWriter, print.LogFailure, "The daprd process exited with error code: %s", daprdErr.Error())
		} else {
			print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogSuccess, "Exited Dapr successfully")
		}
	}()

	if runConfig.AppPort <= 0 {
		// If app does not listen to port, we can check for Dapr's sidecar health before starting the app.
		// Otherwise, it creates a deadlock.
		sidecarUp := true

		if runConfig.UnixDomainSocket != "" {
			httpSocket := utils.GetSocket(runConfig.UnixDomainSocket, runE.AppID, "http")
			print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, "Checking if Dapr sidecar is listening on HTTP socket %v", httpSocket)
			err = utils.IsDaprListeningOnSocket(httpSocket, time.Duration(runtimeWaitTimeoutInSeconds)*time.Second)
			if err != nil {
				sidecarUp = false
				print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Dapr sidecar is not listening on HTTP socket: %s", err.Error())
			}

			grpcSocket := utils.GetSocket(runConfig.UnixDomainSocket, runE.AppID, "grpc")
			print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, "Checking if Dapr sidecar is listening on GRPC socket %v", grpcSocket)
			err = utils.IsDaprListeningOnSocket(grpcSocket, time.Duration(runtimeWaitTimeoutInSeconds)*time.Second)
			if err != nil {
				sidecarUp = false
				print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Dapr sidecar is not listening on GRPC socket: %s", err.Error())
			}
		} else {
			print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, "Checking if Dapr sidecar is listening on HTTP port %v", runE.DaprHTTPPort)
			err = utils.IsDaprListeningOnPort(runE.DaprHTTPPort, time.Duration(runtimeWaitTimeoutInSeconds)*time.Second)
			if err != nil {
				sidecarUp = false
				print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Dapr sidecar is not listening on HTTP port: %s", err.Error())
			}

			print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, "Checking if Dapr sidecar is listening on GRPC port %v", runE.DaprGRPCPort)
			err = utils.IsDaprListeningOnPort(runE.DaprGRPCPort, time.Duration(runtimeWaitTimeoutInSeconds)*time.Second)
			if err != nil {
				sidecarUp = false
				print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Dapr sidecar is not listening on GRPC port: %s", err.Error())
			}
		}

		if sidecarUp {
			print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, "Dapr sidecar is up and running.")
		} else {
			print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Dapr sidecar might not be responding.")
		}
	}
	daprRunning <- true
}

// killDaprdProcess is used to kill the Daprd process and return error on failure.
func killDaprdProcess(runE *runExec.RunExec) error {
	err := runE.DaprCMD.Command.Process.Kill()
	if err != nil {
		print.StatusEvent(runE.DaprCMD.ErrorWriter, print.LogFailure, "Error exiting Dapr: %s", err)
		return err
	}
	print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogSuccess, "Exited Dapr successfully")
	return nil
}

// killAppProcess is used to kill the App process and return error on failure.
func killAppProcess(runE *runExec.RunExec) error {
	if runE.AppCMD.Command == nil || runE.AppCMD.Command.Process == nil {
		return nil
	}
	// Check if the process has already exited on its own.
	if runE.AppCMD.Command.ProcessState != nil && runE.AppCMD.Command.ProcessState.Exited() {
		// Process already exited, no need to kill it.
		return nil
	}
	err := runE.AppCMD.Command.Process.Kill()
	if err != nil {
		// If the process already exited on its own
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		print.StatusEvent(runE.DaprCMD.ErrorWriter, print.LogFailure, "Error exiting App: %s", err)
		return err
	}
	print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogSuccess, "Exited App successfully")
	return nil
}

// putCLIProcessIDInMeta puts the CLI process ID in metadata so that it can be used by the CLI to stop the CLI process.
func putCLIProcessIDInMeta(runE *runExec.RunExec, pid int) {
	print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, "Updating metadata for cliPID: %d", pid)
	err := metadata.Put(runE.DaprHTTPPort, "cliPID", strconv.Itoa(pid), runE.AppID, unixDomainSocket)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for cliPID: %s", err.Error())
	}
}

func putAppProcessIDInMeta(runE *runExec.RunExec) {
	print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, "Updating metadata for appPID: %d", runE.AppCMD.Command.Process.Pid)
	err := metadata.Put(runE.DaprHTTPPort, "appPID", strconv.Itoa(runE.AppCMD.Command.Process.Pid), runE.AppID, unixDomainSocket)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for appPID: %s", err.Error())
	}
}

// putAppCommandInMeta puts the app command in metadata so that it can be used by the CLI to stop the app.
func putAppCommandInMeta(runConfig standalone.RunConfig, runE *runExec.RunExec) {
	appCommand := strings.Join(runConfig.Command, " ")
	print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogInfo, "Updating metadata for app command: %s", appCommand)
	err := metadata.Put(runE.DaprHTTPPort, "appCommand", appCommand, runE.AppID, unixDomainSocket)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for appCommand: %s", err.Error())
		return
	}
}

// putRunFilePathInMeta puts the absolute path of run file in metadata so that it can be used by the CLI to stop all apps started by this run file.
func putRunFilePathInMeta(runE *runExec.RunExec, runFilePath string) {
	runFilePath, err := filepath.Abs(runFilePath)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not get absolute path for run file: %s", err.Error())
		return
	}
	err = metadata.Put(runE.DaprHTTPPort, "runTemplatePath", runFilePath, runE.AppID, unixDomainSocket)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for run file path: %s", err.Error())
	}
}

// putRunTemplateNameInMeta puts the name of the run file in metadata so that it can be used by the CLI to stop all apps started by this run file.
func putRunTemplateNameInMeta(runE *runExec.RunExec, runTemplateName string) {
	err := metadata.Put(runE.DaprHTTPPort, "runTemplateName", runTemplateName, runE.AppID, unixDomainSocket)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for run template name: %s", err.Error())
	}
}

// putAppLogFilePathInMeta puts the absolute path of app log file in metadata so that it can be used by the CLI to stop the app.
func putAppLogFilePathInMeta(runE *runExec.RunExec, appLogFilePath string) {
	err := metadata.Put(runE.DaprHTTPPort, "appLogPath", appLogFilePath, runE.AppID, unixDomainSocket)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for app log file path: %s", err.Error())
	}
}

// putDaprLogFilePathInMeta puts the absolute path of Dapr log file in metadata so that it can be used by the CLI to stop the app.
func putDaprLogFilePathInMeta(runE *runExec.RunExec, daprLogFilePath string) {
	err := metadata.Put(runE.DaprHTTPPort, "daprdLogPath", daprLogFilePath, runE.AppID, unixDomainSocket)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for dapr log file path: %s", err.Error())
	}
}

// getRunFilePath returns the path to the run file.
// If the provided path is a path to a YAML file then return the same.
// Else it returns the path of "dapr.yaml" in the provided directory.
func getRunFilePath(path string) (string, error) {
	if path == "-" {
		return path, nil // will be read from stdin later.
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("error getting file info for %s: %w", path, err)
	}
	if fileInfo.IsDir() {
		filePath, err := utils.FindFileInDir(path, defaultRunTemplateFileName)
		if err != nil {
			return "", err
		}
		return filePath, nil
	}
	hasYAMLExtension := strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
	if !hasYAMLExtension {
		return "", fmt.Errorf("file %q is not a YAML file", path)
	}
	return path, nil
}

// getConflictingFlags checks if any flags are set other than the ones passed in the excludedFlags slice.
// Used for logic or notifications when any of the flags are conflicting and should not be used together.
func getConflictingFlags(cmd *cobra.Command, excludedFlags ...string) []string {
	var conflictingFlags []string
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if !slices.Contains(excludedFlags, f.Name) {
			conflictingFlags = append(conflictingFlags, f.Name)
		}
	})
	return conflictingFlags
}

// detectIncompatibleFlags checks if any incompatible flags are used with --run-file
// and returns a slice of the flag names that were used
func detectIncompatibleFlags(cmd *cobra.Command) []string {
	if runFilePath == "" {
		return nil // No run file specified, so no incompatibilities
	}

	// Get all flags that are not in the compatible list
	return getConflictingFlags(cmd, append(runFileCompatibleFlags, "run-file")...)
}
