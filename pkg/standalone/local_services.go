/*
Copyright 2026 The Dapr Authors
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

package standalone

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	path_filepath "path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/nightlyone/lockfile"

	"github.com/dapr/cli/pkg/print"
)

const (
	// localServiceListenAddress is the address that the local services listen on.
	// They serve this machine only.
	localServiceListenAddress = "127.0.0.1"
	// localServiceHost is the host that daprd uses to connect to the local services.
	localServiceHost = daprDefaultHost

	localServiceStartTimeout = 30 * time.Second
	localServiceStopTimeout  = 10 * time.Second

	// localServicesLockFileName is the lock file in the logs folder. It keeps
	// two `dapr run` commands from starting the same service at the same moment.
	localServicesLockFileName = "local-services.lock"

	daprVersionsWithListenAddressFlags = ">= 1.14.x"

	// placementRaftPort is the raft port of the placement service. The
	// service uses it with its default flags.
	placementRaftPort = 8201
	// schedulerEtcdPeerPort is the etcd peer port of the scheduler service.
	// The service uses it with its default flags.
	schedulerEtcdPeerPort = 2380
)

// localServiceSpec describes how `dapr run` starts one service of a slim installation.
type localServiceSpec struct {
	// name is the binary name of the service. The log file uses the same name.
	name string
	// port is the gRPC port that daprd connects to.
	port        int
	healthzPort int
	// otherPorts are the other ports that the service listens on, on
	// localServiceListenAddress. The service cannot start when another
	// process uses one of them.
	otherPorts []int
	args       []string
}

// localService is a service process that `dapr run` started.
type localService struct {
	spec    localServiceSpec
	logPath string
	cmd     *exec.Cmd
	exited  chan struct{}
	waitErr error
}

// LocalServices starts the placement and scheduler services of a slim
// installation when `dapr run` needs them, and stops them when `dapr run` exits.
//
// A slim installation has the placement and scheduler binaries in the Dapr bin
// folder, but nothing runs them. A Docker installation has no such binaries, so
// LocalServices does nothing for it.
type LocalServices struct {
	binDir           string
	logsDir          string
	schedulerDataDir string
	runtimeVersion   string

	started []*localService
	// handled holds the names of the services that Start already looked at.
	handled map[string]bool

	// Replaced in tests.
	isListening   func(address string) bool
	canListen     func(address string) bool
	out           io.Writer
	placementPort int
	schedulerPort int
	lockTimeout   time.Duration
}

// NewLocalServices returns the local services of the Dapr installation in
// daprInstallPath, which is the value of --runtime-path. The runtime version
// comes from the installed daprd binary.
func NewLocalServices(daprInstallPath string) (*LocalServices, error) {
	installDir, err := GetDaprRuntimePath(daprInstallPath)
	if err != nil {
		return nil, err
	}
	// GetRuntimeVersion returns an error and "n/a" when daprd is not installed
	// or does not run. Then the version is empty, and no service starts.
	version, err := GetRuntimeVersion(daprInstallPath)
	if err != nil {
		version = ""
	}

	return &LocalServices{
		binDir:           getDaprBinPath(installDir),
		logsDir:          getDaprLogsPath(installDir),
		schedulerDataDir: getDaprSchedulerDataPath(installDir),
		runtimeVersion:   strings.TrimSpace(version),
		handled:          map[string]bool{},
		isListening:      isListening,
		canListen:        canListen,
		out:              os.Stdout,
		placementPort:    defaultPlacementPort(),
		schedulerPort:    defaultSchedulerPort(),
		lockTimeout:      localServiceStartTimeout,
	}, nil
}

// Start starts the placement service (when startPlacement is true) and the
// scheduler service (when startScheduler is true) of a slim installation.
// The caller sets a flag to false when the user gave an address for that service.
// A service starts only when its binary is installed, no process accepts
// connections on its default address, and no other process uses one of the
// other ports of the service. Start waits until the services are
// healthy. A service that does not start gives a warning, not an error, because
// daprd can run without it.
//
// Start holds a lock file from the first check until the services are healthy.
// A second `dapr run` that starts at the same moment waits for the lock, and
// then uses the services of the first one.
func (l *LocalServices) Start(startPlacement, startScheduler bool) {
	if l == nil {
		return
	}
	specs := l.plan(startPlacement, startScheduler)
	if len(specs) == 0 {
		return
	}
	unlock := l.lock()
	defer unlock()

	var starting []*localService
	for _, spec := range specs {
		address := localServiceHost + ":" + strconv.Itoa(spec.port)
		if l.isListening(address) {
			print.InfoStatusEvent(l.out, "Using the %s service that runs on %s.", spec.name, address)
			continue
		}
		// With a port in use, the service does not become healthy. The
		// scheduler retries its etcd ports until the start timeout.
		if port, inUse := l.portInUse(spec); inUse {
			l.warnStartFailed(spec, address, fmt.Sprintf("Did not start the %s service, because another process uses port %d.", spec.name, port))
			continue
		}

		logPath := path_filepath.Join(l.logsDir, spec.name+".log")
		s, err := startLocalService(binaryFilePathWithDir(l.binDir, spec.name), logPath, spec)
		if err != nil {
			l.warnStartFailed(spec, address, fmt.Sprintf("Could not start the %s service: %s", spec.name, err))
			continue
		}
		starting = append(starting, s)
	}

	for _, s := range starting {
		address := localServiceHost + ":" + strconv.Itoa(s.spec.port)
		if err := s.waitHealthy(localServiceStartTimeout); err != nil {
			s.stop(localServiceStopTimeout)
			l.warnStartFailed(s.spec, address, fmt.Sprintf("Could not start the %s service on %s: %s. See %s", s.spec.name, address, err, s.logPath))
			continue
		}
		l.started = append(l.started, s)
		print.InfoStatusEvent(l.out, "Started the %s service on %s. It stops when this command exits. Logs: %s", s.spec.name, address, s.logPath)
	}
}

// warnStartFailed reports a service that did not start. Another `dapr run` can
// start the same service at the same moment. Then this start fails because the
// port is in use, and the service of the other command serves the address.
func (l *LocalServices) warnStartFailed(spec localServiceSpec, address, warning string) {
	if l.isListening(address) {
		print.InfoStatusEvent(l.out, "Using the %s service that runs on %s.", spec.name, address)
		return
	}
	print.WarningStatusEvent(l.out, "%s", warning)
}

// lock takes the lock file of the installation, and returns the function that
// releases it. The file holds the process ID, so a lock file of a process that
// no longer runs does not block. lock waits up to lockTimeout for another
// process to release the lock. After that time, or when the lock file does not
// work, Start continues without the lock.
func (l *LocalServices) lock() func() {
	noLock := func() {}
	if err := os.MkdirAll(l.logsDir, 0o755); err != nil {
		return noLock
	}
	path, err := path_filepath.Abs(path_filepath.Join(l.logsDir, localServicesLockFileName))
	if err != nil {
		return noLock
	}
	lockFile, err := lockfile.New(path)
	if err != nil {
		return noLock
	}

	deadline := time.Now().Add(l.lockTimeout)
	for {
		err = lockFile.TryLock()
		if err == nil {
			return func() { _ = lockFile.Unlock() }
		}
		var temporary interface{ Temporary() bool }
		if !errors.As(err, &temporary) || !temporary.Temporary() {
			return noLock
		}
		if time.Now().After(deadline) {
			print.WarningStatusEvent(l.out, "Another dapr run holds %s for more than %s. This command starts the services without the lock.", path, l.lockTimeout)
			return noLock
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// portInUse returns the first of the other ports of spec that another process uses.
func (l *LocalServices) portInUse(spec localServiceSpec) (int, bool) {
	for _, port := range spec.otherPorts {
		if !l.canListen(net.JoinHostPort(localServiceListenAddress, strconv.Itoa(port))) {
			return port, true
		}
	}
	return 0, false
}

// plan returns the services that Start must look at.
func (l *LocalServices) plan(startPlacement, startScheduler bool) []localServiceSpec {
	if l.runtimeVersion == "" {
		return nil
	}
	placementInstalled := fileExists(binaryFilePathWithDir(l.binDir, placementServiceFilePrefix))
	schedulerInstalled := fileExists(binaryFilePathWithDir(l.binDir, schedulerServiceFilePrefix))

	var specs []localServiceSpec
	if startPlacement && placementInstalled && !l.handled[placementServiceFilePrefix] {
		l.handled[placementServiceFilePrefix] = true
		specs = append(specs, placementServiceSpec(l.runtimeVersion, l.placementPort))
	}
	if startScheduler && schedulerInstalled && !l.handled[schedulerServiceFilePrefix] {
		l.handled[schedulerServiceFilePrefix] = true
		// `dapr init --slim --scheduler-placement` installs no placement binary.
		// Then the scheduler also serves actor placement.
		placementEnabled := false
		if !placementInstalled {
			placementEnabled, _ = isSchedulerPlacementIncluded(l.runtimeVersion)
		}
		specs = append(specs, schedulerServiceSpec(l.runtimeVersion, l.schedulerDataDir, l.schedulerPort, placementEnabled))
	}
	return specs
}

// Stop stops the services that Start started, in reverse order.
func (l *LocalServices) Stop() {
	if l == nil {
		return
	}
	for i := len(l.started) - 1; i >= 0; i-- {
		s := l.started[i]
		s.stop(localServiceStopTimeout)
		print.SuccessStatusEvent(l.out, "Stopped the %s service", s.spec.name)
	}
	l.started = nil
}

// SchedulerWarning returns a warning when no process accepts connections on the
// scheduler address that daprd uses. It returns an empty string when the address
// is empty, when the address answers, or when the runtime has no scheduler.
func (l *LocalServices) SchedulerWarning(address *string) string {
	if l == nil || address == nil || strings.TrimSpace(*address) == "" {
		return ""
	}
	if hasScheduler, err := isSchedulerIncluded(l.runtimeVersion); err != nil || !hasScheduler {
		return ""
	}
	addr := strings.TrimSpace(*address)
	if l.isListening(addr) {
		return ""
	}

	msg := fmt.Sprintf("No scheduler service answers on %s. Workflows, actor reminders and the Jobs API do not work until a scheduler service runs there.", addr)
	if !fileExists(binaryFilePathWithDir(l.binDir, schedulerServiceFilePrefix)) {
		msg += " Make sure that the " + DaprSchedulerContainerName + " container runs, or run `dapr init`."
	}
	return msg
}

// placementServiceSpec returns the arguments for a local placement service.
// It uses the same host ports as the placement container of `dapr init`.
func placementServiceSpec(runtimeVersion string, port int) localServiceSpec {
	args := []string{
		"--port", strconv.Itoa(port),
		"--healthz-port", strconv.Itoa(healthPort),
		"--metrics-port", strconv.Itoa(metricPort),
	}
	if hasListenAddressFlags(runtimeVersion) {
		args = append(args,
			"--listen-address", localServiceListenAddress,
			"--healthz-listen-address", localServiceListenAddress,
			"--metrics-listen-address", localServiceListenAddress,
		)
	}
	return localServiceSpec{
		name:        placementServiceFilePrefix,
		port:        port,
		healthzPort: healthPort,
		otherPorts:  []int{placementRaftPort, healthPort, metricPort},
		args:        args,
	}
}

// schedulerServiceSpec returns the arguments for a local scheduler service.
// It uses the same host ports as the scheduler container of `dapr init`.
func schedulerServiceSpec(runtimeVersion, dataDir string, port int, placementEnabled bool) localServiceSpec {
	args := []string{
		"--port", strconv.Itoa(port),
		"--listen-address", localServiceListenAddress,
		"--healthz-port", strconv.Itoa(schedulerHealthPort),
		"--healthz-listen-address", localServiceListenAddress,
		"--metrics-port", strconv.Itoa(schedulerMetricPort),
		"--metrics-listen-address", localServiceListenAddress,
		"--etcd-data-dir", dataDir,
	}
	if schedulerOverrideHostPort(initInfo{runtimeVersion: runtimeVersion}) {
		args = append(args, fmt.Sprintf("--override-broadcast-host-port=%s:%d", localServiceHost, port))
	}
	if placementEnabled {
		args = append(args, "--placement-enabled=true")
	}
	return localServiceSpec{
		name:        schedulerServiceFilePrefix,
		port:        port,
		healthzPort: schedulerHealthPort,
		otherPorts:  []int{schedulerEtcdPort, schedulerEtcdPeerPort, schedulerHealthPort, schedulerMetricPort},
		args:        args,
	}
}

// hasListenAddressFlags returns true if the placement service of the runtime
// version has the --listen-address flags.
func hasListenAddressFlags(runtimeVersion string) bool {
	if runtimeVersion == "edge" || runtimeVersion == "dev" {
		return true
	}
	c, err := semver.NewConstraint(daprVersionsWithListenAddressFlags)
	if err != nil {
		return false
	}
	v, err := semver.NewVersion(runtimeVersion)
	if err != nil {
		return true
	}
	vNoPrerelease, err := v.SetPrerelease("")
	if err != nil {
		return true
	}
	return c.Check(&vNoPrerelease)
}

func defaultPlacementPort() int {
	if runtime.GOOS == daprWindowsOS {
		return 6050
	}
	return 50005
}

func defaultSchedulerPort() int {
	if runtime.GOOS == daprWindowsOS {
		return schedulerWindowsGRPCHostPort
	}
	return 50006
}

// startLocalService starts the binary with the arguments of spec. The process
// appends its output to logPath. It does not truncate the file, because a
// service that another `dapr run` started can still write to it.
func startLocalService(binPath, logPath string, spec localServiceSpec) (*localService, error) {
	if err := os.MkdirAll(path_filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(binPath, spec.args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err = cmd.Start(); err != nil {
		logFile.Close()
		return nil, err
	}

	s := &localService{
		spec:    spec,
		logPath: logPath,
		cmd:     cmd,
		exited:  make(chan struct{}),
	}
	go func() {
		s.waitErr = cmd.Wait()
		logFile.Close()
		close(s.exited)
	}()
	return s, nil
}

// waitHealthy waits until the healthz endpoint of the service returns a 2xx status.
func (s *localService) waitHealthy(timeout time.Duration) error {
	url := fmt.Sprintf("http://%s:%d/healthz", localServiceListenAddress, s.spec.healthzPort)
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-s.exited:
			if s.waitErr != nil {
				return fmt.Errorf("the process exited: %w", s.waitErr)
			}
			return errors.New("the process exited")
		default:
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("not healthy after %s", timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// stop asks the process to exit, and kills it after the timeout.
func (s *localService) stop(timeout time.Duration) {
	select {
	case <-s.exited:
		return
	default:
	}

	if runtime.GOOS == daprWindowsOS {
		_ = s.cmd.Process.Kill()
	} else {
		_ = s.cmd.Process.Signal(os.Interrupt)
	}

	select {
	case <-s.exited:
	case <-time.After(timeout):
		_ = s.cmd.Process.Kill()
		<-s.exited
	}
}

// canListen returns true if this process can listen on the TCP address. Then
// a service that listens on the same address does not fail because the port
// is in use.
func canListen(address string) bool {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// isListening returns true if a process accepts TCP connections on address.
func isListening(address string) bool {
	conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
