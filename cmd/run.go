// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/metadata"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var appPort int
var profilePort int
var appID string
var configFile string
var port int
var grpcPort int
var maxConcurrency int
var image string
var enableProfiling bool
var logLevel string
var protocol string
var componentsPath string

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Launches Dapr and (optionally) your app side by side",
	Long: `Runs Dapr's sidecar and (optionally) an application.

Run a Java application:
  dapr run --app-id myapp -- java -jar myapp.jar
Run a NodeJs application that listens to port 3000:
  dapr run --app-id myapp --app-port 3000 -- node myapp.js
Run a Python application:
  dapr run --app-id myapp -- python myapp.py
Run sidecar only:
  dapr run --app-id myapp
	`,
	Args: cobra.MinimumNArgs(0),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("placement-host", cmd.Flags().Lookup("placement-host"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println(print.WhiteBold("WARNING: no application command found."))
		}

		if kubernetesMode {
			output, err := kubernetes.Run(&kubernetes.RunConfig{
				AppID:         appID,
				AppPort:       appPort,
				GRPCPort:      grpcPort,
				HTTPPort:      port,
				Arguments:     args,
				Image:         image,
				CodeDirectory: args[0],
			})
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}

			print.InfoStatusEvent(os.Stdout, output.Message)
		} else {
			output, err := standalone.Run(&standalone.RunConfig{
				AppID:           appID,
				AppPort:         appPort,
				HTTPPort:        port,
				GRPCPort:        grpcPort,
				ConfigFile:      configFile,
				Arguments:       args,
				EnableProfiling: enableProfiling,
				ProfilePort:     profilePort,
				LogLevel:        logLevel,
				MaxConcurrency:  maxConcurrency,
				Protocol:        protocol,
				PlacementHost:   viper.GetString("placement-host"),
				ComponentsPath:  componentsPath,
			})
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}

			var sigCh = make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

			daprRunning := make(chan bool, 1)
			appRunning := make(chan bool, 1)

			go func() {
				print.InfoStatusEvent(
					os.Stdout,
					fmt.Sprintf(
						"Starting Dapr with id %s. HTTP Port: %v. gRPC Port: %v",
						output.AppID,
						output.DaprHTTPPort,
						output.DaprGRPCPort))

				stdErrPipe, pipeErr := output.DaprCMD.StderrPipe()
				if pipeErr != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stderr for Dapr: %s", err.Error()))
					os.Exit(1)
				}

				stdOutPipe, pipeErr := output.DaprCMD.StdoutPipe()
				if pipeErr != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stdout for Dapr: %s", err.Error()))
					os.Exit(1)
				}

				errScanner := bufio.NewScanner(stdErrPipe)
				outScanner := bufio.NewScanner(stdOutPipe)
				go func() {
					for errScanner.Scan() {
						fmt.Println(print.Yellow(fmt.Sprintf("== DAPR == %s\n", errScanner.Text())))
					}
				}()

				go func() {
					for outScanner.Scan() {
						fmt.Println(print.Yellow(fmt.Sprintf("== DAPR == %s\n", outScanner.Text())))
					}
				}()

				err = output.DaprCMD.Start()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, err.Error())
					os.Exit(1)
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
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stderr for App: %s", err.Error()))
					os.Exit(1)
				}

				stdOutPipe, pipeErr := output.AppCMD.StdoutPipe()
				if pipeErr != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stdout for App: %s", err.Error()))
					os.Exit(1)
				}

				errScanner := bufio.NewScanner(stdErrPipe)
				outScanner := bufio.NewScanner(stdOutPipe)
				go func() {
					for errScanner.Scan() {
						fmt.Println(print.Blue(fmt.Sprintf("== APP == %s\n", errScanner.Text())))
					}
				}()

				go func() {
					for outScanner.Scan() {
						fmt.Println(print.Blue(fmt.Sprintf("== APP == %s\n", outScanner.Text())))
					}
				}()

				err = output.AppCMD.Start()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, err.Error())
					os.Exit(1)
				}

				appRunning <- true
			}()

			<-appRunning

			// Metadata API is only available if app has started listening to port, so wait for app to start before calling metadata API.
			err = metadata.Put(output.DaprHTTPPort, "cliPID", strconv.Itoa(os.Getpid()))
			if err != nil {
				print.WarningStatusEvent(os.Stdout, "Could not update sidecar metadata for cliPID: %s", err.Error())
			}

			if output.AppCMD != nil {
				appCommand := strings.Join(args, " ")
				print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Updating metadata for app command: %s", appCommand))
				err = metadata.Put(output.DaprHTTPPort, "appCommand", appCommand)
				if err != nil {
					print.WarningStatusEvent(os.Stdout, "Could not update sidecar metadata for appCommand: %s", err.Error())
				}

				print.SuccessStatusEvent(os.Stdout, "You're up and running! Both Dapr and your app logs will appear here.\n")
			} else {
				print.SuccessStatusEvent(os.Stdout, "You're up and running! Dapr logs will appear here.\n")
			}

			<-sigCh
			print.InfoStatusEvent(os.Stdout, "\nterminated signal received: shutting down")

			err = output.DaprCMD.Process.Kill()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error exiting Dapr: %s", err))
			} else {
				print.SuccessStatusEvent(os.Stdout, "Exited Dapr successfully")
			}

			if output.AppCMD != nil {
				err = output.AppCMD.Process.Kill()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error exiting App: %s", err))
				} else {
					print.SuccessStatusEvent(os.Stdout, "Exited App successfully")
				}
			}
		}
	},
}

func init() {
	RunCmd.Flags().IntVarP(&appPort, "app-port", "p", -1, "the port your application is listening on")
	RunCmd.Flags().StringVarP(&appID, "app-id", "i", "", "an id for your application, used for service discovery")
	RunCmd.Flags().StringVarP(&configFile, "config", "c", standalone.DefaultConfigFilePath(), "Dapr configuration file. Default is $HOME/.dapr/config.yaml or %USERPROFILE%\\.dapr\\config.yaml")
	RunCmd.Flags().IntVarP(&port, "dapr-http-port", "H", -1, "the HTTP port for Dapr to listen on")
	RunCmd.Flags().IntVarP(&grpcPort, "dapr-grpc-port", "G", -1, "the gRPC port for Dapr to listen on")
	RunCmd.Flags().StringVarP(&image, "image", "", "", "the image to build the code in. input is repository/image")
	RunCmd.Flags().BoolVar(&enableProfiling, "enable-profiling", false, "Enable pprof profiling via an HTTP endpoint")
	RunCmd.Flags().IntVarP(&profilePort, "profile-port", "", -1, "the port for the profile server to listen on")
	RunCmd.Flags().StringVarP(&logLevel, "log-level", "", "info", "Sets the log verbosity. Valid values are: debug, info, warn, error, fatal, or panic. Default is info")
	RunCmd.Flags().IntVarP(&maxConcurrency, "max-concurrency", "", -1, "controls the concurrency level of the app. Default is unlimited")
	RunCmd.Flags().StringVarP(&protocol, "app-protocol", "P", "http", "tells Dapr to use HTTP or gRPC to talk to the app. Default is http")
	RunCmd.Flags().StringVarP(&componentsPath, "components-path", "d", standalone.DefaultComponentsDirPath(), "Path for components directory. Default is $HOME/.dapr/components or %USERPROFILE%\\.dapr\\components")
	RunCmd.Flags().String("placement-host-address", "localhost", "the host on which the placement service resides")

	// deprecated flags
	RunCmd.Flags().IntVarP(&port, "port", "", -1, "the HTTP port for Dapr to listen on")
	RunCmd.Flags().IntVarP(&grpcPort, "grpc-port", "", -1, "the gRPC port for Dapr to listen on")
	RunCmd.Flags().String("placement-host", "localhost", "the host on which the placement service resides")
	RunCmd.Flags().StringVarP(&protocol, "protocol", "", "http", "tells Dapr to use HTTP or gRPC to talk to the app. Default is http")

	RunCmd.Flags().MarkDeprecated("port", "this flag is deprecated and will be removed in v1.0. Use dapr-http-port instead")
	RunCmd.Flags().MarkDeprecated("grpc-port", "this flag is deprecated and will be removed in v1.0. Use dapr-grpc-port instead")
	RunCmd.Flags().MarkDeprecated("placement-host", "this flag is deprecated and will be removed in v1.0. Use placement-host-address instead")
	RunCmd.Flags().MarkDeprecated("protocol", "this flag is deprecated and will be removed in v1.0. Use app-protocol instead")

	RootCmd.AddCommand(RunCmd)
}
