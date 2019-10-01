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

const (
	componentsDirName           = "components"
	redisMessageBusYamlFileName = "redis_messagebus.yaml"
	redisStateStoreYamlFileName = "redis.yaml"
)

type RunConfig struct {
	AppID           string
	AppPort         int
	Port            int
	ConfigFile      string
	Arguments       []string
	EnableProfiling bool
	ProfilePort     int
	LogLevel        string
	MaxConcurrency  int
}

type RunOutput struct {
	ActionsCMD  *exec.Cmd
	ActionsPort int
	AppID       string
	AppCMD      *exec.Cmd
}

type component struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Type     string                  `yaml:"type"`
		Metadata []componentMetadataItem `yaml:"metadata"`
	} `yaml:"spec"`
}

type componentMetadataItem struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func getActionsCommand(appID string, actionsPort int, appPort int, configFile string, enableProfiling bool, profilePort int, logLevel string, maxConcurrency int) (*exec.Cmd, int, error) {
	if actionsPort < 0 {
		port, err := freeport.GetFreePort()
		if err != nil {
			return nil, -1, err
		}

		actionsPort = port
	}

	if maxConcurrency < 1 {
		maxConcurrency = -1
	}

	actionsCMD := "actionsrt"
	if runtime.GOOS == "windows" {
		actionsCMD = fmt.Sprintf("%s.exe", actionsCMD)
	}

	args := []string{"--actions-id", appID, "--actions-http-port", fmt.Sprintf("%v", actionsPort), "--log-level", logLevel, "--max-concurrency", fmt.Sprintf("%v", maxConcurrency)}
	if appPort > -1 {
		args = append(args, "--app-port")
		args = append(args, fmt.Sprintf("%v", appPort))
	}

	args = append(args, "--placement-address")

	if runtime.GOOS == "windows" {
		args = append(args, "localhost:6050")
	} else {
		args = append(args, "localhost:50005")
	}

	args = append(args, "--actions-grpc-port")
	grpcPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, -1, err
	}

	args = append(args, fmt.Sprintf("%v", grpcPort))

	if configFile != "" {
		args = append(args, "--config")
		args = append(args, configFile)
	}

	if enableProfiling {
		if profilePort == -1 {
			profilePort, err = freeport.GetFreePort()
			if err != nil {
				return nil, -1, err
			}
		}

		args = append(args, "--enable-profiling")
		args = append(args, "true")
		args = append(args, "--profile-port")
		args = append(args, fmt.Sprintf("%v", profilePort))
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

func dirOrFileExists(dirOrFilePath string) bool {
	_, err := os.Stat(dirOrFilePath)
	return !os.IsNotExist(err)
}

func absoluteComponentsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return path.Join(wd, componentsDirName), nil
}

func createRedisStateStore() error {
	redisStore := component{
		APIVersion: "actions.io/v1alpha1",
		Kind:       "Component",
	}

	redisStore.Metadata.Name = "statestore"
	redisStore.Spec.Type = "state.redis"
	redisStore.Spec.Metadata = []componentMetadataItem{}
	redisStore.Spec.Metadata = append(redisStore.Spec.Metadata, componentMetadataItem{
		Name:  "redisHost",
		Value: "localhost:6379",
	})
	redisStore.Spec.Metadata = append(redisStore.Spec.Metadata, componentMetadataItem{
		Name:  "redisPassword",
		Value: "",
	})

	b, err := yaml.Marshal(&redisStore)
	if err != nil {
		return err
	}

	componentsDir, err := absoluteComponentsDir()
	if err != nil {
		return err
	}

	os.Mkdir(componentsDir, 0777)
	err = ioutil.WriteFile(path.Join(componentsDir, redisStateStoreYamlFileName), b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func createRedisPubSub() error {
	redisMessageBus := component{
		APIVersion: "actions.io/v1alpha1",
		Kind:       "Component",
	}

	redisMessageBus.Metadata.Name = "messagebus"
	redisMessageBus.Spec.Type = "pubsub.redis"
	redisMessageBus.Spec.Metadata = []componentMetadataItem{}
	redisMessageBus.Spec.Metadata = append(redisMessageBus.Spec.Metadata, componentMetadataItem{
		Name:  "redisHost",
		Value: "localhost:6379",
	})
	redisMessageBus.Spec.Metadata = append(redisMessageBus.Spec.Metadata, componentMetadataItem{
		Name:  "redisPassword",
		Value: "",
	})

	b, err := yaml.Marshal(&redisMessageBus)
	if err != nil {
		return err
	}

	componentsDir, err := absoluteComponentsDir()
	if err != nil {
		return err
	}

	os.Mkdir(componentsDir, 0777)
	err = ioutil.WriteFile(path.Join(componentsDir, redisMessageBusYamlFileName), b, 0644)
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

	actions, err := List()
	if err != nil {
		return nil, err
	}

	for _, a := range actions {
		if appID == a.AppID {
			return nil, fmt.Errorf("actions with ID %s is already running", appID)
		}
	}

	componentsDir, err := absoluteComponentsDir()
	if err != nil {
		return nil, err
	}

	if !dirOrFileExists(path.Join(componentsDir, redisStateStoreYamlFileName)) {
		err = createRedisStateStore()
		if err != nil {
			return nil, err
		}
	}

	if !dirOrFileExists(path.Join(componentsDir, redisMessageBusYamlFileName)) {
		err = createRedisPubSub()
		if err != nil {
			return nil, err
		}
	}

	actionsCMD, actionsPort, err := getActionsCommand(appID, config.Port, config.AppPort, config.ConfigFile, config.EnableProfiling, config.ProfilePort, config.LogLevel, config.MaxConcurrency)
	if err != nil {
		return nil, err
	}

	for _, a := range actions {
		if actionsPort == a.ActionsPort {
			return nil, fmt.Errorf("there's already an actions instance running with port %v", actionsPort)
		}
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
