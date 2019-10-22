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
	"strings"
	"syscall"
	"time"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/rundata"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
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

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Launches dapr and your app side by side",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		uuid, err := uuid.NewRandom()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			return
		}

		daprRunID := uuid.String()

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
			})
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}

			var sigCh = make(chan os.Signal)
			signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

			daprRunning := make(chan bool, 1)
			appRunning := make(chan bool, 1)
			daprRunCreatedTime := time.Now()

			go func() {
				print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Starting Dapr with id %s. HTTP Port: %v. gRPC Port: %v", output.AppID, output.DaprHTTPPort, output.DaprGRPCPort))

				stdErrPipe, err := output.DaprCMD.StderrPipe()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stderr for Dapr: %s", err.Error()))
					os.Exit(1)
				}

				stdOutPipe, err := output.DaprCMD.StdoutPipe()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stdout for Dapr: %s", err.Error()))
					os.Exit(1)
				}

				errScanner := bufio.NewScanner(stdErrPipe)
				outScanner := bufio.NewScanner(stdOutPipe)
				go func() {
					//pure client app wait till daprd ready for conn
					start_app := false
					ready_msg := "dapr initialized. Status: Running"
					for errScanner.Scan() {
						text := errScanner.Text()
						fmt.Printf(print.Yellow(fmt.Sprintf("== DAPR == %s\n", text)))
						//stop searching after app started
						if appPort == -1 && !start_app &&
							strings.Index(text, ready_msg) != -1 {
							//daprd is ready, start appCmd
							start_app = true
							daprRunning <- true
						}
					}
				}()

				go func() {
					for outScanner.Scan() {
						fmt.Printf(print.Yellow(fmt.Sprintf("== DAPR == %s\n", outScanner.Text())))
					}
				}()

				err = output.DaprCMD.Start()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, err.Error())
					os.Exit(1)
				}

				//if app exposes service, don't wait for daprd
				if appPort != -1 {
					daprRunning <- true
				}

			}()

			<-daprRunning

			go func() {
				stdErrPipe, err := output.AppCMD.StderrPipe()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stderr for App: %s", err.Error()))
					os.Exit(1)
				}

				stdOutPipe, err := output.AppCMD.StdoutPipe()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stdout for App: %s", err.Error()))
					os.Exit(1)
				}

				errScanner := bufio.NewScanner(stdErrPipe)
				outScanner := bufio.NewScanner(stdOutPipe)
				go func() {
					for errScanner.Scan() {
						fmt.Printf(print.Blue(fmt.Sprintf("== APP == %s\n", errScanner.Text())))
					}
				}()

				go func() {
					for outScanner.Scan() {
						fmt.Printf(print.Blue(fmt.Sprintf("== APP == %s\n", outScanner.Text())))
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

			rundata.AppendRunData(&rundata.RunData{
				DaprRunId:    daprRunID,
				AppId:        output.AppID,
				DaprHTTPPort: output.DaprHTTPPort,
				DaprGRPCPort: output.DaprGRPCPort,
				AppPort:      appPort,
				Command:      strings.Join(args, " "),
				Created:      daprRunCreatedTime,
				PID:          os.Getpid(),
			})

			print.SuccessStatusEvent(os.Stdout, "You're up and running! Both Dapr and your app logs will appear here.\n")

			<-sigCh
			print.InfoStatusEvent(os.Stdout, "\nterminated signal received: shutting down")

			rundata.ClearRunData(daprRunID)

			err = output.DaprCMD.Process.Kill()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error exiting Dapr: %s", err))
			} else {
				print.SuccessStatusEvent(os.Stdout, "Exited Dapr successfully")
			}

			err = output.AppCMD.Process.Kill()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error exiting App: %s", err))
			} else {
				print.SuccessStatusEvent(os.Stdout, "Exited App successfully")
			}
		}
	},
}

func init() {
	RunCmd.Flags().IntVarP(&appPort, "app-port", "", -1, "the port your application is listening on")
	RunCmd.Flags().StringVarP(&appID, "app-id", "", "", "an id for your application, used for service discovery")
	RunCmd.Flags().StringVarP(&configFile, "config", "", "", "Dapr configuration file")
	RunCmd.Flags().IntVarP(&port, "port", "p", -1, "the HTTP port for Dapr to listen on")
	RunCmd.Flags().IntVarP(&grpcPort, "grpc-port", "", -1, "the gRPC port for Dapr to listen on")
	RunCmd.Flags().StringVarP(&image, "image", "", "", "the image to build the code in. input is repository/image")
	RunCmd.Flags().BoolVar(&enableProfiling, "enable-profiling", false, "Enable pprof profiling via an HTTP endpoint")
	RunCmd.Flags().IntVarP(&profilePort, "profile-port", "", -1, "the port for the profile server to listen on")
	RunCmd.Flags().BoolVar(&kubernetesMode, "kubernetes", false, "build and deploy your app and Dapr to a Kubernetes cluster")
	RunCmd.Flags().StringVarP(&logLevel, "log-level", "", "info", "Sets the log verbosity. Valid values are: debug, info, warning, error, fatal, or panic. Default is info")
	RunCmd.Flags().IntVarP(&maxConcurrency, "max-concurrency", "", -1, "controls the concurrency level of the app. Default is unlimited")
	RunCmd.Flags().StringVarP(&protocol, "protocol", "", "http", "tells Dapr to use HTTP or gRPC to talk to the app. Default is http")

	RootCmd.AddCommand(RunCmd)
}
