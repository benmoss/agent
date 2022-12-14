package kubernetes

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/buildkite/agent/v3/logger"
	"github.com/buildkite/agent/v3/process"
)

func init() {
	gob.Register(new(syscall.WaitStatus))
}

const (
	defaultSocketPath = "/workspace/buildkite.sock"

	checkoutContainerID = 0
	commandContainerID  = 1
)

func New(l logger.Logger, c Config) *Runner {
	if c.SocketPath == "" {
		c.SocketPath = defaultSocketPath
	}
	clients := make(map[int]*clientResult, c.ClientCount)
	for i := 0; i < c.ClientCount; i++ {
		clients[i] = &clientResult{}
	}
	return &Runner{
		logger:  l,
		conf:    c,
		clients: clients,
		server:  rpc.NewServer(),
		mux:     http.NewServeMux(),
		done:    make(chan struct{}),
		started: make(chan struct{}),
	}
}

type Runner struct {
	logger   logger.Logger
	conf     Config
	mu       sync.Mutex
	listener net.Listener
	started,
	done,
	interrupt chan struct{}
	startedOnce,
	closedOnce,
	interruptOnce sync.Once
	server  *rpc.Server
	mux     *http.ServeMux
	clients map[int]*clientResult
}

type clientResult struct {
	ExitStatus process.WaitStatus
	State      clientState
}

type clientState int

const (
	stateUnknown clientState = iota
	stateConnected
	stateExited
)

func (c clientResult) Connected() bool {
	return c.State == stateConnected || c.State == stateExited
}

func (c clientResult) Exited() bool {
	return c.State == stateExited
}

type Config struct {
	SocketPath     string
	ClientCount    int
	Stdout, Stderr io.Writer
	AccessToken    string
}

// Starts the Runner, listening for RPC messages on the socket
func (r *Runner) Run(ctx context.Context) error {
	r.server.Register(r)
	r.mux.Handle(rpc.DefaultRPCPath, r.server)

	l, err := (&net.ListenConfig{}).Listen(ctx, "unix", r.conf.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer l.Close()
	defer os.Remove(r.conf.SocketPath)
	r.listener = l
	go http.Serve(l, r.mux)

	<-r.done
	r.logger.Debug("runner done")
	return nil
}

// Returns whether the Runner has been started
func (r *Runner) Started() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.started
}

// Returns whether the Runner has completed
func (r *Runner) Done() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.done
}

// Interrupts all clients, triggering graceful shutdown
func (r *Runner) Interrupt() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.interruptOnce.Do(func() {
		close(r.interrupt)
	})
	return nil
}

// Stops the RPC server, allowing Run to return immediately
func (r *Runner) Terminate() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closedOnce.Do(func() {
		close(r.done)
	})
	return nil
}

func (r *Runner) WaitStatus() process.WaitStatus {
	// if bootstrap failed, return that
	bootstrap := r.clients[checkoutContainerID]
	if bootstrap.ExitStatus != nil && bootstrap.ExitStatus.ExitStatus() != 0 {
		return bootstrap.ExitStatus
	}

	// otherwise return command's exit
	return r.clients[commandContainerID].ExitStatus
}

// ==== sidecar api ====

type Empty struct{}
type Logs struct {
	Data []byte
}

type ExitCode struct {
	ID         int
	ExitStatus process.WaitStatus
}

type RegisterResponse struct {
	AccessToken string
}

type RunState string

const (
	RunStateWait      RunState = "wait"
	RunStateGo        RunState = "go"
	RunStateInterrupt RunState = "interrupt"
)

func (r *Runner) WriteLogs(args Logs, reply *Empty) error {
	_, err := io.Copy(r.conf.Stdout, bytes.NewReader(args.Data))
	return err
}

func (r *Runner) Exit(args ExitCode, reply *Empty) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, found := r.clients[args.ID]
	if !found {
		return fmt.Errorf("unrecognized client id: %d", args.ID)
	}
	r.logger.Info("client %d exited with code %d", args.ID, args.ExitStatus.ExitStatus())
	client.ExitStatus = args.ExitStatus
	client.State = stateExited

	allExited := true
	for _, client := range r.clients {
		allExited = client.State == stateExited && allExited
	}
	if allExited {
		r.closedOnce.Do(func() {
			close(r.done)
		})
	}
	return nil
}

func (r *Runner) Register(id int, reply *RegisterResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.startedOnce.Do(func() {
		close(r.started)
	})
	client, found := r.clients[id]
	if !found {
		return fmt.Errorf("client id %d not found", id)
	}
	if client.Connected() {
		return fmt.Errorf("client id %d already registered", id)
	}
	r.logger.Info("client %d connected", id)
	client.State = stateConnected
	reply.AccessToken = r.conf.AccessToken
	return nil
}

func (r *Runner) Status(id int, reply *RunState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case <-r.done:
		return rpc.ErrShutdown
	case <-r.interrupt:
		*reply = RunStateInterrupt
		return nil
	default:
		// continue
	}

	switch id {
	case checkoutContainerID:
		*reply = RunStateGo
	case commandContainerID:
		ready := true
	Out:
		for id, client := range r.clients {
			switch id {
			case commandContainerID:
				continue
			case checkoutContainerID:
				if !client.Exited() {
					ready = false
					break Out
				}
			default:
				if !client.Connected() {
					ready = false
					break Out
				}
			}
		}
		if ready {
			*reply = RunStateGo
		} else {
			*reply = RunStateWait
		}
	default:
		if _, found := r.clients[id]; found {
			if r.clients[commandContainerID].Exited() {
				*reply = RunStateInterrupt
			} else if r.clients[checkoutContainerID].Exited() {
				*reply = RunStateGo
			} else {
				*reply = RunStateWait
			}
		} else {
			return fmt.Errorf("client id %d not found", id)
		}
	}
	r.logger.Info("client %d ping, state: %s", id, *reply)
	return nil
}

type Client struct {
	ID         int
	SocketPath string
	client     *rpc.Client
}

var errNotConnected = errors.New("client not connected")

func (c *Client) Connect() (RegisterResponse, error) {
	if c.SocketPath == "" {
		c.SocketPath = defaultSocketPath
	}
	client, err := rpc.DialHTTP("unix", c.SocketPath)
	if err != nil {
		return RegisterResponse{}, err
	}
	c.client = client
	var resp RegisterResponse
	if err := c.client.Call("Runner.Register", c.ID, &resp); err != nil {
		return RegisterResponse{}, err
	}
	return resp, nil
}

func (c *Client) Exit(exitStatus process.WaitStatus) error {
	if c.client == nil {
		return errNotConnected
	}
	return c.client.Call("Runner.Exit", ExitCode{
		ID:         c.ID,
		ExitStatus: exitStatus,
	}, nil)
}

// Write implements io.Writer
func (c *Client) Write(p []byte) (int, error) {
	if c.client == nil {
		return 0, errNotConnected
	}
	if c.ID != checkoutContainerID && c.ID != commandContainerID {
		return 0, nil
	}
	n := len(p)
	err := c.client.Call("Runner.WriteLogs", Logs{
		Data: p,
	}, nil)
	return n, err
}

func (c *Client) AwaitRunState(desiredState RunState) error {
	for {
		var current RunState
		if err := c.client.Call("Runner.Status", c.ID, &current); err != nil {
			if desiredState == RunStateInterrupt && errors.Is(err, rpc.ErrShutdown) {
				return nil
			}
			return err
		}
		if current == desiredState {
			return nil
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (c *Client) Close() {
	c.client.Close()
}

func (c *Client) IsSidecar() bool {
	return c.ID > commandContainerID
}
