package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/actionscore/cli/pkg/kubernetes"
	"github.com/actionscore/cli/pkg/print"
	"github.com/actionscore/cli/pkg/rundata"
	"github.com/actionscore/cli/pkg/standalone"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var appPort int
var appID string
var port int
var image string

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Launches Actions and your app side by side",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		uuid, err := uuid.NewRandom()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			return
		}

		actionsRunID := uuid.String()

		if kubernetesMode {
			output, err := kubernetes.Run(&kubernetes.RunConfig{
				AppID:         appID,
				AppPort:       appPort,
				Port:          port,
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
				AppID:     appID,
				AppPort:   appPort,
				Port:      port,
				Arguments: args,
			})
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}

			var sigCh = make(chan os.Signal)
			signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

			actionsRunning := make(chan bool, 1)
			appRunning := make(chan bool, 1)
			actionsRunCreatedTime := time.Now()

			go func() {
				print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Starting Actions with id %s on port %v", output.AppID, output.ActionsPort))

				stdErrPipe, err := output.ActionsCMD.StderrPipe()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stderr for Actions: %s", err.Error()))
					os.Exit(1)
				}

				stdOutPipe, err := output.ActionsCMD.StdoutPipe()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error creating stdout for Actions: %s", err.Error()))
					os.Exit(1)
				}

				errScanner := bufio.NewScanner(stdErrPipe)
				outScanner := bufio.NewScanner(stdOutPipe)
				go func() {
					for errScanner.Scan() {
						fmt.Printf(print.Yellow(fmt.Sprintf("== ACTIONS == %s\n", errScanner.Text())))
					}
				}()

				go func() {
					for outScanner.Scan() {
						fmt.Printf(print.Yellow(fmt.Sprintf("== ACTIONS == %s\n", outScanner.Text())))
					}
				}()

				err = output.ActionsCMD.Start()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, err.Error())
					os.Exit(1)
				}

				actionsRunning <- true
			}()

			<-actionsRunning

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
				ActionsRunId: actionsRunID,
				AppId:        output.AppID,
				ActionsPort:  output.ActionsPort,
				AppPort:      appPort,
				Command:      strings.Join(args, " "),
				Created:      actionsRunCreatedTime,
			})

			print.SuccessStatusEvent(os.Stdout, "You're up and running! Both Actions and your app logs will appear here.\n")

			<-sigCh
			print.InfoStatusEvent(os.Stdout, "\nterminated signal recieved: shutting down")

			rundata.ClearRunData(actionsRunID)

			err = output.ActionsCMD.Process.Kill()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error exiting Actions: %s", err))
			} else {
				print.SuccessStatusEvent(os.Stdout, "Exited Actions successfully")
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
	RunCmd.Flags().IntVarP(&port, "port", "p", -1, "the port for Actions to listen on")
	RunCmd.Flags().StringVarP(&image, "image", "", "", "the image to build the code in. input is repository/image")
	RunCmd.Flags().BoolVar(&kubernetesMode, "kubernetes", false, "Build and deploy your app and Actions to a Kubernetes cluster")
	RootCmd.AddCommand(RunCmd)
}
