package actor

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

// PID is the process identifier type
type PID int64

var ErrProcessNotRunning = errors.New("process not running")

type Process struct {
	actor   Actor
	mailbox chan any
	stop    chan struct{}
	exited  bool
	err     error
	mu      sync.Mutex
}

func (p *Process) Start(supervisorCtx context.Context) (restart bool) {
	restart = false
	p.mu.Lock()
	defer p.mu.Unlock()

	processCtx, cancel := context.WithCancel(supervisorCtx)
	defer cancel()

	p.exited = false
	defer func() { p.exited = true }()

	p.stop = make(chan struct{})

	for !p.exited {
		select {
		case <-processCtx.Done():
			return
		case <-p.stop:
			return
		case msg := <-p.mailbox:
			ctx, cancel := context.WithCancel(processCtx)
			c := NewContext(ctx, cancel, p.actor)
			p.actor.Receive(c, msg)
			<-c.Done()
			err := c.Err()
			if err != nil {
				p.err = err
				restart = true
				return
			}
		}
	}

	return
}

func (p *Process) Stop() error {
	if p.exited {
		return ErrProcessNotRunning
	}

	close(p.stop)
	return nil
}

func (p *Process) Send(msg any) error {
	if p.exited {
		return ErrProcessNotRunning
	}
	p.mailbox <- msg

	return nil
}

// StartFunc is a factory function type for starting Actors through the Supervisor
type StartFunc func(PID, Dispatcher) (Actor, error)

var ErrNoProcessWithPID = errors.New("no process with the given pid")

// Dispatcher an interface that allows the sending of messages between actor processes
type Dispatcher interface {
	Send(PID, any) error
	// TODO: An address model would be pretty cool so you don't have to know the PIDs of all the actors you want to talk to
}

// Supervisor supervices Actors
type Supervisor struct {
	ctx    context.Context
	cancel context.CancelFunc
	// TODO: probably replace this with a supervisor tree in the future
	processes map[PID]*Process
}

// NewSupervisor creates a new instance of Supervisor
func NewSupervisor(parentCtx context.Context) *Supervisor {
	ctx, cancel := context.WithCancel(parentCtx)
	return &Supervisor{ctx, cancel, make(map[PID]*Process)}
}

func (s *Supervisor) pidExists(pid PID) bool {
	_, exists := s.processes[pid]
	return exists
}

// Start starts an Actor with a given PID
func (s *Supervisor) Start(startFunc StartFunc) (PID, error) {
	rand.Seed(time.Now().UnixNano())

	// generate pid for actor
	pid := PID(rand.Int63())

	for s.pidExists(pid) {
		pid = PID(rand.Int63())
	}

	actor, err := startFunc(pid, s)
	if err != nil {
		return -1, err
	}

	proc := &Process{actor: actor, mailbox: make(chan any)}
	s.processes[pid] = proc

	go s.startProcess(pid)

	return pid, nil
}

func (s *Supervisor) startProcess(pid PID) {
	proc, _ := s.processes[pid]

	// will keep restarting it so long as the process wants to, probably need to make this configurable at some point...
	restart := true
	for restart {
		restart = proc.Start(s.ctx)
	}
}

// only stops, but doesn't remove the process
func (s *Supervisor) softStop(pid PID) error {
	proc, exists := s.processes[pid]
	if !exists {
		return ErrNoProcessWithPID
	}

	return proc.Stop()
}

// Stop stops a running Actor
func (s *Supervisor) Stop(pid PID) error {
	err := s.softStop(pid)
	if err != nil {
		return err
	}

	// remove the process
	delete(s.processes, pid)
	return nil
}

// Send sends a message to the Actor at a given PID
func (s *Supervisor) Send(pid PID, msg any) error {
	proc, exists := s.processes[pid]
	if !exists {
		return ErrNoProcessWithPID
	}

	return proc.Send(msg)
}
