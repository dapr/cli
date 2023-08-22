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

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	appV1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8s "k8s.io/client-go/kubernetes"
	podsv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// Specifically use k8s sig yaml to marshal into json, then convert to yaml.
	k8sYaml "sigs.k8s.io/yaml"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone/runfileconfig"
	daprsyscall "github.com/dapr/cli/pkg/syscall"
	"github.com/dapr/cli/utils"
)

const (
	serviceKind             = "Service"
	deploymentKind          = "Deployment"
	serviceAPIVersion       = "v1"
	deploymentAPIVersion    = "apps/v1"
	loadBalanceType         = "LoadBalancer"
	daprEnableAnnotationKey = "dapr.io/enabled"
	serviceFileName         = "service.yaml"
	deploymentFileName      = "deployment.yaml"
	appLabelKey             = "app"
	nameKey                 = "name"
	labelsKey               = "labels"
	tcpProtocol             = "TCP"

	podCreationDeletionTimeout = 1 * time.Minute
)

type deploymentConfig struct {
	Kind       string               `json:"kind"`
	APIVersion string               `json:"apiVersion"`
	Metadata   map[string]any       `json:"metadata"`
	Spec       appV1.DeploymentSpec `json:"spec"`
}

type serviceConfig struct {
	Kind       string             `json:"kind"`
	APIVersion string             `json:"apiVersion"`
	Metadata   map[string]any     `json:"metadata"`
	Spec       corev1.ServiceSpec `json:"spec"`
}

type runState struct {
	serviceFilePath    string
	deploymentFilePath string
	app                runfileconfig.App
	logCancel          context.CancelFunc
}

// Run executes the application based on the run file configuration.
// Run creates a temporary `deploy` folder within the app/.dapr directory and then applies that to the context pointed to
// kubectl client.
func Run(runFilePath string, config runfileconfig.RunFileConfig) (bool, error) {
	// At this point, we expect the runfile to be parsed and the values within config
	// Validations and default setting will only be done after this point.
	var exitWithError bool

	// get k8s client PodsInterface.
	client, cErr := Client()
	if cErr != nil {
		// exit with error.
		return true, fmt.Errorf("error getting k8s client: %w", cErr)
	}

	namespace := corev1.NamespaceDefault
	podsInterface := client.CoreV1().Pods(namespace)

	// setup a monitoring context for shutdown call from another cli process.
	monitoringContext, monitoringCancel := context.WithCancel(context.Background())
	defer monitoringCancel()

	// setup shutdown notify channel.
	sigCh := make(chan os.Signal, 1)
	daprsyscall.SetupShutdownNotify(sigCh)

	runStates := []runState{}

	for _, app := range config.Apps {
		print.StatusEvent(os.Stdout, print.LogInfo, "Validating config and starting app %q", app.RunConfig.AppID)
		// Set defaults if zero value provided in config yaml.
		app.RunConfig.SetDefaultFromSchema()

		// Validate validates the configs for k8s and modifies appId etc.
		err := app.RunConfig.ValidateK8s()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error validating run config for app %q present in %s: %s", app.RunConfig.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}

		var svc serviceConfig
		// create default service config.
		if app.ContainerConfiguration.CreateService {
			svc = createServiceConfig(app)
		}

		// create default deployment config.
		dep := createDeploymentConfig(app)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error creating deployment file for app %q present in %s: %s", app.RunConfig.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}
		// overwrite <app-id>/.dapr/deploy/service.yaml.
		// overwrite <app-id>/.dapr/deploy/deployment.yaml.

		err = writeYamlFile(app, svc, dep)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error creating deployment/service yaml files: %s", err.Error())
			exitWithError = true
			break
		}

		deployDir := app.GetDeployDir()
		print.InfoStatusEvent(os.Stdout, "Deploying app %q to Kubernetes", app.AppID)
		serviceFilePath := filepath.Join(deployDir, serviceFileName)
		deploymentFilePath := filepath.Join(deployDir, deploymentFileName)
		rState := runState{}
		if app.CreateService {
			print.InfoStatusEvent(os.Stdout, "Deploying service YAML %q to Kubernetes", serviceFilePath)
			err = deployYamlToK8s(serviceFilePath)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Error deploying service yaml file %q : %s", serviceFilePath, err.Error())
				exitWithError = true
				break
			}
			rState.serviceFilePath = serviceFilePath
		}

		print.InfoStatusEvent(os.Stdout, "Deploying deployment YAML %q to Kubernetes", deploymentFilePath)
		err = deployYamlToK8s(deploymentFilePath)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error deploying deployment yaml file %q : %s", deploymentFilePath, err.Error())
			exitWithError = true
			break
		}

		// create log files and save state.
		err = app.CreateDaprdLogFile()
		if err != nil {
			print.StatusEvent(os.Stderr, print.LogFailure, "Error getting daprd log file for app %q present in %s: %s", app.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}
		err = app.CreateAppLogFile()
		if err != nil {
			print.StatusEvent(os.Stderr, print.LogFailure, "Error getting app log file for app %q present in %s: %s", app.AppID, runFilePath, err.Error())
			exitWithError = true
			break
		}

		daprdLogWriter := runfileconfig.GetLogWriter(app.DaprdLogWriteCloser, app.DaprdLogDestination)
		// appDaprdWriter := runExec.GetAppDaprdWriter(app, false).
		appLogWriter := runfileconfig.GetLogWriter(app.AppLogWriteCloser, app.AppLogDestination)

		ctx, cancel := context.WithTimeout(context.Background(), podCreationDeletionTimeout)
		err = waitPodRunning(ctx, client, namespace, app.AppID)
		cancel()
		if err != nil {
			print.WarningStatusEvent(os.Stderr, "Error deploying pod to Kubernetes. See logs directly from Kubernetes command line.")
			// Close the log files since there is deployment error, and the container might be in crash loop back off state.
			app.CloseAppLogFile()
			app.CloseDaprdLogFile()
		} else {
			logContext, cancel := context.WithCancel(context.Background())
			rState.logCancel = cancel
			err = setupLogs(logContext, app.AppID, daprdLogWriter, appLogWriter, podsInterface)
			if err != nil {
				print.StatusEvent(os.Stderr, print.LogWarning, "Error setting up logs for app %q present in %q . See logs directly from Kubernetes command line.: %s ", app.AppID, runFilePath, err.Error())
			}
		}

		rState.deploymentFilePath = deploymentFilePath
		rState.app = app

		// append runSate only on successful k8s deploy.
		runStates = append(runStates, rState)

		print.InfoStatusEvent(os.Stdout, "Writing log files to directory : %s", app.GetLogsDir())
	}

	// If all apps have been started and there are no errors in starting the apps wait for signal from sigCh.
	if !exitWithError {
		print.InfoStatusEvent(os.Stdout, "Starting to monitor Kubernetes pods for deletion.")
		go monitorK8sPods(monitoringContext, client, namespace, runStates, sigCh)
		// After all apps started wait for sigCh.
		<-sigCh
		monitoringCancel()
		print.InfoStatusEvent(os.Stdout, "Stopping Kubernetes pods monitoring.")
		// To add a new line in Stdout.
		fmt.Println()
		print.InfoStatusEvent(os.Stdout, "Received signal to stop. Deleting K8s Dapr app deployments.")
	}

	closeErr := gracefullyShutdownK8sDeployment(runStates, client, namespace)
	return exitWithError, closeErr
}

func createServiceConfig(app runfileconfig.App) serviceConfig {
	return serviceConfig{
		Kind:       serviceKind,
		APIVersion: serviceAPIVersion,
		Metadata: map[string]any{
			nameKey: app.RunConfig.AppID,
			labelsKey: map[string]string{
				appLabelKey: app.AppID,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   tcpProtocol,
					Port:       80,
					TargetPort: intstr.FromInt(app.AppPort),
				},
			},
			Selector: map[string]string{
				appLabelKey: app.AppID,
			},
			Type: loadBalanceType,
		},
	}
}

func createDeploymentConfig(app runfileconfig.App) deploymentConfig {
	replicas := int32(1)
	dep := deploymentConfig{
		Kind:       deploymentKind,
		APIVersion: deploymentAPIVersion,
		Metadata: map[string]any{
			nameKey: app.AppID,
		},
	}

	dep.Spec = appV1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				appLabelKey: app.AppID,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					appLabelKey: app.AppID,
				},
				Annotations: app.RunConfig.GetAnnotations(),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            app.AppID,
						Image:           app.ContainerImage,
						Env:             getEnv(app),
						ImagePullPolicy: corev1.PullAlways,
					},
				},
			},
		},
	}
	// Set dapr.io/enable annotation.
	dep.Spec.Template.ObjectMeta.Annotations[daprEnableAnnotationKey] = "true"

	// set containerPort only if app port is present.
	if app.AppPort != 0 {
		dep.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				ContainerPort: int32(app.AppPort),
			},
		}
	}

	return dep
}

func getEnv(app runfileconfig.App) []corev1.EnvVar {
	envs := app.GetEnv()
	envVars := make([]corev1.EnvVar, len(envs))
	i := 0
	for k, v := range app.GetEnv() {
		envVars[i] = corev1.EnvVar{
			Name:  k,
			Value: v,
		}
		i++
	}
	return envVars
}

func writeYamlFile(app runfileconfig.App, svc serviceConfig, dep deploymentConfig) error {
	var yamlBytes []byte
	var err error
	var writeFile io.WriteCloser
	deployDir := app.GetDeployDir()
	if app.CreateService {
		yamlBytes, err = k8sYaml.Marshal(svc)
		if err != nil {
			return fmt.Errorf("error marshalling service yaml: %w", err)
		}
		serviceFilePath := filepath.Join(deployDir, serviceFileName)
		writeFile, err = os.Create(serviceFilePath)
		if err != nil {
			return fmt.Errorf("error creating file %s : %w", serviceFilePath, err)
		}
		_, err = writeFile.Write(yamlBytes)
		if err != nil {
			writeFile.Close()
			return fmt.Errorf("error writing to file %s : %w", serviceFilePath, err)
		}
		writeFile.Close()
	}
	yamlBytes, err = k8sYaml.Marshal(dep)
	if err != nil {
		return fmt.Errorf("error marshalling deployment yaml: %w", err)
	}
	deploymentFilePath := filepath.Join(deployDir, deploymentFileName)
	writeFile, err = os.Create(deploymentFilePath)
	if err != nil {
		return fmt.Errorf("error creating file %s : %w", deploymentFilePath, err)
	}
	_, err = writeFile.Write(yamlBytes)
	if err != nil {
		writeFile.Close()
		return fmt.Errorf("error writing to file %s : %w", deploymentFilePath, err)
	}
	writeFile.Close()
	return nil
}

func deployYamlToK8s(yamlToDeployPath string) error {
	_, err := utils.RunCmdAndWait("kubectl", "apply", "-f", yamlToDeployPath)
	if err != nil {
		return fmt.Errorf("error deploying the yaml %s to Kubernetes: %w", yamlToDeployPath, err)
	}
	return nil
}

func deleteYamlK8s(yamlToDeletePath string) error {
	print.InfoStatusEvent(os.Stdout, "Deleting %q from Kubernetes", yamlToDeletePath)
	_, err := utils.RunCmdAndWait("kubectl", "delete", "-f", yamlToDeletePath)
	if err != nil {
		return fmt.Errorf("error deploying the yaml %s to Kubernetes: %w", yamlToDeletePath, err)
	}
	return nil
}

func setupLogs(ctx context.Context, appID string, daprdLogWriter, appLogWriter io.Writer, podInterface podsv1.PodInterface) error {
	return streamContainerLogsToDisk(ctx, appID, appLogWriter, daprdLogWriter, podInterface)
}

func gracefullyShutdownK8sDeployment(runStates []runState, client k8s.Interface, namespace string) error {
	errs := make([]error, 0, len(runStates)*4)
	for _, r := range runStates {
		if len(r.serviceFilePath) != 0 {
			errs = append(errs, deleteYamlK8s(r.serviceFilePath))
		}
		errs = append(errs, deleteYamlK8s(r.deploymentFilePath))
		labelSelector := map[string]string{
			daprAppIDKey: r.app.AppID,
		}
		if ok, _ := CheckPodExists(client, namespace, labelSelector, r.app.AppID); ok {
			ctx, cancel := context.WithTimeout(context.Background(), podCreationDeletionTimeout)
			err := waitPodDeleted(ctx, client, namespace, r.app.AppID)
			cancel()
			if err != nil {
				// swallowing err here intentionally.
				print.WarningStatusEvent(os.Stderr, "Error waiting for pods to be deleted. Final logs might only be partially available.")
			}
		}

		// shutdown logs.
		r.logCancel()
		errs = append(errs, r.app.CloseAppLogFile(), r.app.CloseDaprdLogFile())
	}
	return errors.Join(errs...)
}

func monitorK8sPods(ctx context.Context, client k8s.Interface, namespace string, runStates []runState, sigCh chan os.Signal) {
	// for each app wait for pod to be deleted, if all pods are deleted, then send shutdown signal to the cli process.

	wg := sync.WaitGroup{}

	for _, r := range runStates {
		go func(appID string, wg *sync.WaitGroup) {
			err := waitPodDeleted(ctx, client, namespace, r.app.AppID)
			if err != nil && strings.Contains(err.Error(), podWatchErrTemplate) {
				print.WarningStatusEvent(os.Stderr, "Error monitoring Kubernetes pod(s) for app %q.", appID)
			}
			wg.Done()
		}(r.app.AppID, &wg)
		wg.Add(1)
	}
	wg.Wait()
	// Send signal to gracefully close log writers and shut down process.
	sigCh <- syscall.SIGINT
}
