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

type component struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Type           string            `json:"type"`
		ConnectionInfo map[string]string `json:"connectionInfo"`
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

	actionsCMD := "actionsrt"
	if runtime.GOOS == "windows" {
		actionsCMD = fmt.Sprintf("%s.exe", actionsCMD)
	}

	args := []string{"--actions-id", appID, "--actions-http-port", fmt.Sprintf("%v", actionsPort)}
	if appPort > -1 {
		args = append(args, "--app-port")
		args = append(args, fmt.Sprintf("%v", appPort))
	}

	args = append(args, "--placement-address")

	if runtime.GOOS == "windows" {
		args = append(args, "localhost:6050")
		args = append(args, "--actions-grpc-port", "6051")
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

func createRedisStateStore() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	redisStore := component{
		APIVersion: "actions.io/v1alpha1",
		Kind:       "Component",
	}

	redisStore.Metadata.Name = "statestore"
	redisStore.Spec.Type = "state.redis"
	redisStore.Spec.ConnectionInfo = map[string]string{}
	redisStore.Spec.ConnectionInfo["redisHost"] = "localhost:6379"
	redisStore.Spec.ConnectionInfo["redisPassword"] = ""

	b, err := yaml.Marshal(&redisStore)
	if err != nil {
		return err
	}

	os.Mkdir(path.Join(wd, "components"), 0777)
	err = ioutil.WriteFile(path.Join(path.Join(wd, "components"), "redis.yaml"), b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func createRedisPubSub() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	redisMessageBus := component{
		APIVersion: "actions.io/v1alpha1",
		Kind:       "Component",
	}

	redisMessageBus.Metadata.Name = "messagebus"
	redisMessageBus.Spec.Type = "pubsub.redis"
	redisMessageBus.Spec.ConnectionInfo = map[string]string{}
	redisMessageBus.Spec.ConnectionInfo["redisHost"] = "localhost:6379"
	redisMessageBus.Spec.ConnectionInfo["password"] = ""

	b, err := yaml.Marshal(&redisMessageBus)
	if err != nil {
		return err
	}

	os.Mkdir(path.Join(wd, "components"), 0777)
	err = ioutil.WriteFile(path.Join(path.Join(wd, "components"), "redis_messagebus.yaml"), b, 0644)
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

	err := createRedisStateStore()
	if err != nil {
		return nil, err
	}

	err = createRedisPubSub()
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
