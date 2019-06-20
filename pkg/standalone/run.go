package standalone

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/Pallinder/sillyname-go"
	"github.com/phayes/freeport"
)

type RunConfig struct {
	AppID     string
	AppPort   int
	Port      int
	Arguments []string
}

type RunOutput struct {
	ActionsCMD  *exec.Cmd
	ActionsPort int
	AppID       string
	AppCMD      *exec.Cmd
}

type eventSource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Type           string `json:"type"`
		ConnectionInfo struct {
			RedisHost     string `json:"redisHost"`
			RedisPassword string `json:"redisPassword"`
		} `json:"connectionInfo"`
	} `json:"spec"`
}

func getActionsCommand(appID string, actionsPort int, appPort int) (*exec.Cmd, int, error) {
	if actionsPort < 0 {
		port, err := freeport.GetFreePort()
		if err != nil {
			return nil, -1, err
		}

		actionsPort = port
	}

	actionsCMD := "action"
	if runtime.GOOS == "windows" {
		actionsCMD = actionsCMD + ".exe"
	}

	args := []string{"--action-id", appID, "--action-http-port", fmt.Sprintf("%v", actionsPort)}
	if appPort > -1 {
		args = append(args, "--app-port")
		args = append(args, fmt.Sprintf("%v", appPort))
	}

	args = append(args, "--assigner-address")

	if runtime.GOOS == "windows" {
		args = append(args, "localhost:6050")
		args = append(args, "--action-grpc-port", "6051")
	} else {
		args = append(args, "localhost:50005")
	}

	cmd := exec.Command(actionsCMD, args...)
	return cmd, actionsPort, nil
}

func getAppCommand(actionsPort int, command string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("ACTIONS_PORT=%v", actionsPort))

	return cmd, nil
}

func createStateEventSource() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	es := eventSource{
		APIVersion: "actions.io/v1alpha1",
		Kind:       "EventSource",
	}

	es.Metadata.Name = "statestore"
	es.Spec.Type = "actions.state.redis"
	es.Spec.ConnectionInfo.RedisHost = "localhost:6379"
	es.Spec.ConnectionInfo.RedisPassword = ""

	b, err := yaml.Marshal(&es)
	if err != nil {
		return err
	}

	os.Mkdir(path.Join(wd, "eventsources"), 0777)

	err = ioutil.WriteFile(path.Join(path.Join(wd, "eventsources"), "redis.yaml"), b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func Run(config *RunConfig) (*RunOutput, error) {
	appID := config.AppID
	if appID == "" {
		appID = strings.Replace(sillyname.GenerateStupidName(), " ", "-", -1)
	}

	err := createStateEventSource()
	if err != nil {
		return nil, err
	}

	actionsCMD, actionsPort, err := getActionsCommand(appID, config.Port, config.AppPort)
	if err != nil {
		return nil, err
	}

	runArgs := []string{}
	argCount := len(config.Arguments)

	if argCount == 0 {
		return nil, errors.New("No app entrypoint given")
	}

	cmd := config.Arguments[0]
	if len(config.Arguments) > 1 {
		runArgs = config.Arguments[1:]
	}

	appCMD, err := getAppCommand(actionsPort, cmd, runArgs)
	if err != nil {
		return nil, err
	}

	return &RunOutput{
		ActionsCMD:  actionsCMD,
		AppCMD:      appCMD,
		AppID:       appID,
		ActionsPort: actionsPort,
	}, nil
}
