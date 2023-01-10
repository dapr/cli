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
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dapr/cli/pkg/metadata"
	"github.com/dapr/cli/pkg/print"
	runExec "github.com/dapr/cli/pkg/run_exec"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/pkg/standalone/runfileconfig"
	"github.com/dapr/cli/utils"
)

var (
	appPort            int
	profilePort        int
	appID              string
	configFile         string
	port               int
	grpcPort           int
	internalGRPCPort   int
	maxConcurrency     int
	enableProfiling    bool
	logLevel           string
	protocol           string
	componentsPath     string
	resourcesPath      string
	appSSL             bool
	metricsPort        int
	maxRequestBodySize int
	readBufferSize     int
	unixDomainSocket   string
	enableAppHealth    bool
	appHealthPath      string
	appHealthInterval  int
	appHealthTimeout   int
	appHealthThreshold int
	enableAPILogging   bool
	apiListenAddresses string
	runFilePath        string
)

const (
	runtimeWaitTimeoutInSeconds = 60
)

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

# Run sidecar only specifying dapr runtime installation directory
dapr run --app-id myapp --dapr-path /usr/local/dapr
  `,
	Args: cobra.MinimumNArgs(0),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("placement-host-address", cmd.Flags().Lookup("placement-host-address"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(runFilePath) > 0 {
			executeRunWithAppsConfigFile(runFilePath)
			return
		}
		if len(args) == 0 {
			fmt.Println(print.WhiteBold("WARNING: no application command found."))
		}

		daprDirPath, err := standalone.GetDaprPath(daprPath)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Failed to get Dapr install directory: %v", err)
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
			if runtime.GOOS == "windows" {
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
			PlacementHostAddr:  viper.GetString("placement-host-address"),
			ComponentsPath:     componentsPath,
			ResourcesPath:      resourcesPath,
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
		}
		output, err := runExec.NewOutput(&standalone.RunConfig{
			AppID:            appID,
			AppPort:          appPort,
			HTTPPort:         port,
			GRPCPort:         grpcPort,
			ProfilePort:      profilePort,
			Command:          args,
			MetricsPort:      metricsPort,
			UnixDomainSocket: unixDomainSocket,
			InternalGRPCPort: internalGRPCPort,
			DaprPathCmdFlag:  daprPath,
			SharedRunConfig:  *sharedRunConfig,
		})
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}

		sigCh := make(chan os.Signal, 1)
		setupShutdownNotify(sigCh)

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
			print.InfoStatusEvent(os.Stdout, startInfo)

			output.DaprCMD.Stdout = os.Stdout
			output.DaprCMD.Stderr = os.Stderr

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

			stdErrPipe, pipeErr := output.AppCMD.StderrPipe()
			if pipeErr != nil {
				print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error creating stderr for App: %s", err.Error()))
				appRunning <- false
				return
			}

			stdOutPipe, pipeErr := output.AppCMD.StdoutPipe()
			if pipeErr != nil {
				print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error creating stdout for App: %s", err.Error()))
				appRunning <- false
				return
			}

			errScanner := bufio.NewScanner(stdErrPipe)
			outScanner := bufio.NewScanner(stdOutPipe)
			go func() {
				for errScanner.Scan() {
					fmt.Println(print.Blue(fmt.Sprintf("== APP == %s", errScanner.Text())))
				}
			}()

			go func() {
				for outScanner.Scan() {
					fmt.Println(print.Blue(fmt.Sprintf("== APP == %s", outScanner.Text())))
				}
			}()

			err = output.AppCMD.Start()
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				appRunning <- false
				return
			}

			go func() {
				appErr := output.AppCMD.Wait()

				if appErr != nil {
					output.AppErr = appErr
					print.FailureStatusEvent(os.Stderr, "The App process exited with error code: %s", appErr.Error())
				} else {
					print.SuccessStatusEvent(os.Stdout, "Exited App successfully")
				}
				sigCh <- os.Interrupt
			}()

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
		err = metadata.Put(output.DaprHTTPPort, "cliPID", strconv.Itoa(os.Getpid()), appID, unixDomainSocket)
		if err != nil {
			print.WarningStatusEvent(os.Stdout, "Could not update sidecar metadata for cliPID: %s", err.Error())
		}

		if output.AppCMD != nil {
			appCommand := strings.Join(args, " ")
			print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Updating metadata for app command: %s", appCommand))
			err = metadata.Put(output.DaprHTTPPort, "appCommand", appCommand, appID, unixDomainSocket)
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
		} else if output.AppCMD != nil && (output.AppCMD.ProcessState == nil || !output.AppCMD.ProcessState.Exited()) {
			err = output.AppCMD.Process.Kill()
			if err != nil {
				exitWithError = true
				print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error exiting App: %s", err))
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
	RunCmd.Flags().StringVarP(&protocol, "app-protocol", "P", "http", "The protocol (gRPC or HTTP) Dapr uses to talk to the application")
	RunCmd.Flags().StringVarP(&componentsPath, "components-path", "d", "", "The path for components directory")
	RunCmd.Flags().StringVarP(&resourcesPath, "resources-path", "", "", "The path for resources directory")
	// TODO: Remove below line once the flag is removed in the future releases.
	// By marking this as deprecated, the flag will be hidden from the help menu, but will continue to work. It will show a warning message when used.
	RunCmd.Flags().MarkDeprecated("components-path", "This flag is deprecated and will be removed in the future releases. Use \"resources-path\" flag instead")
	RunCmd.Flags().String("placement-host-address", "localhost", "The address of the placement service. Format is either <hostname> for default port or <hostname>:<port> for custom port")
	RunCmd.Flags().BoolVar(&appSSL, "app-ssl", false, "Enable https when Dapr invokes the application")
	RunCmd.Flags().IntVarP(&metricsPort, "metrics-port", "M", -1, "The port of metrics on dapr")
	RunCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RunCmd.Flags().IntVarP(&maxRequestBodySize, "dapr-http-max-request-size", "", -1, "Max size of request body in MB")
	RunCmd.Flags().IntVarP(&readBufferSize, "dapr-http-read-buffer-size", "", -1, "HTTP header read buffer in KB")
	RunCmd.Flags().StringVarP(&unixDomainSocket, "unix-domain-socket", "u", "", "Path to a unix domain socket dir. If specified, Dapr API servers will use Unix Domain Sockets")
	RunCmd.Flags().BoolVar(&enableAppHealth, "enable-app-health-check", false, "Enable health checks for the application using the protocol defined with app-protocol")
	RunCmd.Flags().StringVar(&appHealthPath, "app-health-check-path", "", "Path used for health checks; HTTP only")
	RunCmd.Flags().IntVar(&appHealthInterval, "app-health-probe-interval", 0, "Interval to probe for the health of the app in seconds")
	RunCmd.Flags().IntVar(&appHealthTimeout, "app-health-probe-timeout", 0, "Timeout for app health probes in milliseconds")
	RunCmd.Flags().IntVar(&appHealthThreshold, "app-health-threshold", 0, "Number of consecutive failures for the app to be considered unhealthy")
	RunCmd.Flags().BoolVar(&enableAPILogging, "enable-api-logging", false, "Log API calls at INFO verbosity. Valid values are: true or false")
	RunCmd.Flags().StringVar(&apiListenAddresses, "dapr-listen-addresses", "", "Comma separated list of IP addresses that sidecar will listen to")
	RunCmd.Flags().StringVarP(&runFilePath, "run-file", "f", "", "Path to the configuration file for the apps to run")
	RootCmd.AddCommand(RunCmd)
}

func executeRun(runFilePath string, apps []runfileconfig.App) (bool, error) {
	var exitWithError bool
	// setup shutdown notify channel.
	sigCh := make(chan os.Signal, 1)
	setupShutdownNotify(sigCh)

	runStates := make([]*runExec.RunExec, 0, len(apps))
	for _, app := range apps {
		// Get Run Config for different apps
		runConfig := app.RunConfig

		// Validate validates the configs and modifies the ports to free ports, appId etc.
		err := runConfig.Validate()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error validating run config for app %s present in %s: %s", runConfig.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}
		appLogWriter, err := app.GetAppLogFileWriter()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error getting log file for app %s present in %s: %s", runConfig.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}
		daprdLogWriter, err := app.GetDaprdLogFileWriter()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error getting log file for app %s present in %s: %s", runConfig.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}
		runState, err := startDaprdAndAppProcesses(&runConfig, app.AppDirPath, sigCh, daprdLogWriter, daprdLogWriter, appLogWriter, appLogWriter)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "Error starting Dapr and app: %s", err.Error())
			exitWithError = true
			break
		}
		// Store runState in an array
		runStates = append(runStates, runState)
		logInfomationalStatusToStdout(app)
	}
	// If all apps have been started and there are no errors in starting the apps wait for signal from sigCh
	if !exitWithError {
		// After all apps started wait for sigCh
		<-sigCh
	}

	// Stop daprd and app processes for each runState
	exitWithError, closeError := gracefullyShutdownAppsAndCloseResources(runStates, apps)

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

func logInfomationalStatusToStdout(app runfileconfig.App) {
	print.InfoStatusEvent(os.Stdout, "Started Dapr with app id %s. HTTP Port: %d. gRPC Port: %d",
		app.AppID, app.RunConfig.HTTPPort, app.RunConfig.GRPCPort)
	print.InfoStatusEvent(os.Stdout, "Writing log files to directory : %s", app.GetLogsDir())
}

func gracefullyShutdownAppsAndCloseResources(runState []*runExec.RunExec, apps []runfileconfig.App) (bool, error) {
	exitWithError := false
	for _, s := range runState {
		hasErr := stopDaprdAndAppProcesses(s)
		if !exitWithError && hasErr {
			exitWithError = true
		}
	}
	var err error
	// close log file resources
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
	return exitWithError, err
}

func executeRunWithAppsConfigFile(runFilePath string) {
	config := runfileconfig.RunFileConfig{}
	apps, err := config.GetApps(runFilePath)
	if err != nil {
		print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error getting apps from config file: %s", err))
		os.Exit(1)
	}
	if len(apps) == 0 {
		print.FailureStatusEvent(os.Stdout, "No apps to run")
		os.Exit(1)
	}
	exitWithError, closeErr := executeRun(runFilePath, apps)
	if exitWithError {
		if closeErr != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error closing resources: %s", closeErr))
		}
		os.Exit(1)
	}
}

// startDaprdAndAppProcesses is a function to start the App process and the associated Daprd process
// This function also calls metadata API to put CLI process ID and the associated App command
// This should be called as a blocking function call
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
	// appCmd does not need to call SetStdout and SetStderr since output is being read processed and then written
	// to the output and error writers for an app.
	appCmd.WithOutputWriter(appOutputWriter)
	appCmd.WithErrorWriter(appErrorWriter)
	if strings.TrimSpace(commandDir) != "" {
		appCmd.Command.Dir = commandDir
	}

	runState := runExec.New(runConfig, daprCMD, appCmd)

	startErrChan := make(chan error)

	go startDaprdProcess(runConfig, runState, daprRunning, sigCh, startErrChan)

	daprStarted := <-daprRunning
	if !daprStarted {
		startErr := <-startErrChan
		print.StatusEvent(daprdErrorWriter, print.LogFailure, "Error starting daprd process: %s", startErr.Error())
		return nil, err
	}

	go startAppProcess(runConfig, runState, appRunning, sigCh, startErrChan)

	appRunStatus := <-appRunning
	if !appRunStatus {
		// Start App failed, try to stop Dapr and exit.
		err = killDaprdProcess(runState)
		return nil, err
	}

	// Metadata API is only available if app has started listening to port, so wait for app to start before calling metadata API.
	_ = putCLIProcessIDInMeta(runState)

	if runState.AppCMD.Command != nil {
		_ = putAppCommandInMeta(runConfig, runState)
	} else {
		print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogSuccess, "You're up and running! Dapr logs will appear here.\n")
	}
	return runState, nil
}

// stopDaprdAndAppProcesses is a function to stop the App process and the associated Daprd process
// This should be called as a blocking function call
func stopDaprdAndAppProcesses(runState *runExec.RunExec) bool {
	var err error
	print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogInfo, "\ntermination signal received: shutting down")
	// Only if two different output writers are present run the following print statement
	if runState.AppCMD.OutputWriter != runState.DaprCMD.OutputWriter {
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
	} else if runState.AppCMD.Command != nil && (runState.AppCMD.Command.ProcessState == nil || !runState.AppCMD.Command.ProcessState.Exited()) {
		err = killAppProcess(runState)
		if err != nil {
			exitWithError = true
		}
	}
	return exitWithError
}

// startAppsProcess, starts the App process and calls wait in a goroutine
// This function should be called as a goroutine
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
			// Directly output app logs to the error writer only prefixing with == APP ==
			fmt.Fprintln(runE.AppCMD.ErrorWriter, print.Blue(fmt.Sprintf("== APP == %s", errScanner.Text())))
		}
	}()

	go func() {
		for outScanner.Scan() {
			// Directly output app logs to the output writer only prefixing with == APP ==
			fmt.Fprintln(runE.AppCMD.OutputWriter, print.Blue(fmt.Sprintf("== APP == %s", outScanner.Text())))
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
// This function should be called as a goroutine
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
				print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Dapr sidecar is not listening on HTTP port: %s", runE.AppID, err.Error())
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

// killDaprdProcess is used to kill the Daprd process and return error on failure
func killDaprdProcess(runE *runExec.RunExec) error {
	err := runE.DaprCMD.Command.Process.Kill()
	if err != nil {
		print.StatusEvent(runE.DaprCMD.ErrorWriter, print.LogFailure, "Error exiting Dapr: %s", err)
		return err
	}
	print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogSuccess, "Exited Dapr successfully")
	return nil
}

// killAppProcess is used to kill the App process and return error on failure
func killAppProcess(runE *runExec.RunExec) error {
	err := runE.AppCMD.Command.Process.Kill()
	if err != nil {
		print.StatusEvent(runE.DaprCMD.ErrorWriter, print.LogFailure, "Error exiting App: %s", err)
		return err
	}
	print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogSuccess, "Exited App successfully")
	return nil
}

// putCLIProcessIDInMeta puts the CLI process ID in metadata so that it can be used by the CLI to stop the CLI process.
func putCLIProcessIDInMeta(runE *runExec.RunExec) error {
	// For now putting this as 0, since we do not want the dapr stop command for a single to kill the CLI process,
	// thereby killing all the apps that are running via dapr run -f.
	err := metadata.Put(runE.DaprHTTPPort, "cliPID", "0", runE.AppID, unixDomainSocket)
	if err != nil {
		print.StatusEvent(runE.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for cliPID: %s", err.Error())
		return err
	}
	return nil
}

// putAppCommandInMeta puts the app command in metadata so that it can be used by the CLI to stop the app.
func putAppCommandInMeta(runConfig *standalone.RunConfig, runState *runExec.RunExec) error {
	appCommand := strings.Join(runConfig.Command, " ")
	print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogInfo, "Updating metadata for app command: %s", appCommand)
	err := metadata.Put(runState.DaprHTTPPort, "appCommand", appCommand, runState.AppID, runConfig.UnixDomainSocket)
	if err != nil {
		print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogWarning, "Could not update sidecar metadata for appCommand: %s", err.Error())
		return err
	}
	print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogSuccess, "You're up and running! Both Dapr and your app logs will appear here.\n")
	return nil
}
