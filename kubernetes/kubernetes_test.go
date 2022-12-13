package kubernetes

import (
	"context"
	"encoding/gob"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/buildkite/agent/v3/logger"
	"github.com/stretchr/testify/require"
)

func TestOrderedClients(t *testing.T) {
	runner := newRunner(t, 4)
	socketPath := runner.conf.SocketPath

	checkout := &Client{ID: checkoutContainerID, SocketPath: socketPath}
	command := &Client{ID: commandContainerID, SocketPath: socketPath}
	sidecar1 := &Client{ID: 2, SocketPath: socketPath}
	sidecar2 := &Client{ID: 3, SocketPath: socketPath}

	t.Log("waiting for runner to listen")
	require.Eventually(t, func() bool {
		_, err := os.Lstat(socketPath)
		return err == nil

	}, time.Second*10, time.Millisecond, "expected socket file to exist")

	// command should not start until other clients connect
	var runState RunState
	require.NoError(t, runner.Status(command.ID, &runState))
	require.Equal(t, runState, RunStateWait)

	// sidecars should not start until after checkout
	require.NoError(t, runner.Status(sidecar1.ID, &runState))
	require.Equal(t, runState, RunStateWait)
	require.NoError(t, runner.Status(sidecar2.ID, &runState))
	require.Equal(t, runState, RunStateWait)

	// checkout should be ready immediately
	require.NoError(t, runner.Status(checkout.ID, &runState))
	require.Equal(t, runState, RunStateGo)

	// connect checkout
	_, err := checkout.Connect()
	require.NoError(t, err)
	t.Cleanup(checkout.Close)

	// mark checkout exit successful
	require.NoError(t, checkout.Exit(waitStatusSuccess))

	// sidecars should be ready after checkout exits
	require.NoError(t, runner.Status(sidecar1.ID, &runState))
	require.Equal(t, runState, RunStateGo)
	require.NoError(t, runner.Status(sidecar2.ID, &runState))
	require.Equal(t, runState, RunStateGo)

	// connect sidecars
	for _, client := range []*Client{sidecar1, sidecar2} {
		_, err := client.Connect()
		require.NoError(t, err)
		t.Cleanup(client.Close)
	}

	// connect command
	_, err = command.Connect()
	require.NoError(t, err)
	t.Cleanup(command.Close)

	t.Log("command should be ready after sidecar connects")
	require.NoError(t, runner.Status(command.ID, &runState))
	require.Equal(t, runState, RunStateGo)
	require.NoError(t, command.AwaitRunState(RunStateGo))

	// after command exits other clients should be terminated
	require.NoError(t, command.Exit(waitStatusSuccess))

	t.Log("Waiting for sidecar1 to be in RunStateTerminate")
	require.NoError(t, runner.Status(command.ID, &runState))
	require.Equal(t, runState, RunStateGo)
	require.NoError(t, sidecar1.AwaitRunState(RunStateTerminate))

	t.Log("Waiting for sidecar2 to be in RunStateTerminate")
	require.NoError(t, runner.Status(command.ID, &runState))
	require.Equal(t, runState, RunStateGo)
	require.NoError(t, sidecar1.AwaitRunState(RunStateTerminate))
}

func TestDuplicateClients(t *testing.T) {
	runner := newRunner(t, 2)
	socketPath := runner.conf.SocketPath

	client0 := Client{ID: 0, SocketPath: socketPath}
	client1 := Client{ID: 0, SocketPath: socketPath}

	// wait for runner to listen
	require.Eventually(t, func() bool {
		_, err := os.Lstat(socketPath)
		return err == nil

	}, time.Second*10, time.Millisecond, "expected socket file to exist")

	_, err := client0.Connect()
	require.NoError(t, err)
	_, err = client1.Connect()
	require.Error(t, err)
}

func TestExcessClients(t *testing.T) {
	runner := newRunner(t, 1)
	socketPath := runner.conf.SocketPath

	client0 := Client{ID: checkoutContainerID, SocketPath: socketPath}
	client1 := Client{ID: commandContainerID, SocketPath: socketPath}

	_, err := client0.Connect()
	require.NoError(t, err)
	_, err = client1.Connect()
	require.Error(t, err)
}

func TestWaitStatusNonZero(t *testing.T) {
	runner := newRunner(t, 2)

	bootstrap := Client{ID: checkoutContainerID, SocketPath: runner.conf.SocketPath}
	command := Client{ID: commandContainerID, SocketPath: runner.conf.SocketPath}

	_, err := bootstrap.Connect()
	require.NoError(t, err)
	_, err = command.Connect()
	require.NoError(t, err)
	require.NoError(t, bootstrap.Exit(waitStatusFailure))
	require.NoError(t, command.Exit(waitStatusSuccess))
	require.Equal(t, runner.WaitStatus().ExitStatus(), 1)
}

func TestDoneAfterAllClientsExit(t *testing.T) {
	containers := 4
	runner := newRunner(t, containers)
	select {
	case <-runner.Done():
		t.Fatal("runner should not be done")
	default:
		// success
	}
	for i := 0; i < containers; i++ {
		require.NoError(t, runner.Exit(ExitCode{ID: i, ExitStatus: waitStatusSuccess}, nil))
		if i == containers-1 {
			select {
			case <-runner.Done():
				// success
			default:
				t.Fatal("runner should be done")
			}
		} else {
			select {
			case <-runner.Done():
				t.Fatalf("runner should not be done, i: %d", i)
			default:
				// success
			}
		}
	}
}

func newRunner(t *testing.T, clientCount int) *Runner {
	tempDir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	socketPath := filepath.Join(tempDir, "bk.sock")
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	runner := New(logger.Discard, Config{
		SocketPath:  socketPath,
		ClientCount: clientCount,
	})
	runnerCtx, cancelRunner := context.WithCancel(context.Background())
	go runner.Run(runnerCtx)
	t.Cleanup(func() {
		cancelRunner()
	})

	// wait for runner to listen
	require.Eventually(t, func() bool {
		_, err := os.Lstat(socketPath)
		return err == nil

	}, time.Second*10, time.Millisecond, "expected socket file to exist")

	return runner
}

var (
	waitStatusSuccess  = waitStatus{Code: 0}
	waitStatusFailure  = waitStatus{Code: 1}
	waitStatusSignaled = waitStatus{Code: 0, SignalCode: intptr(1)}
)

func init() {
	gob.Register(new(waitStatus))
}

type waitStatus struct {
	Code       int
	SignalCode *int
}

func (w waitStatus) ExitStatus() int {
	return w.Code
}

func (w waitStatus) Signaled() bool {
	return w.SignalCode != nil
}

func (w waitStatus) Signal() syscall.Signal {
	return syscall.Signal(*w.SignalCode)
}

func intptr(x int) *int {
	return &x
}
