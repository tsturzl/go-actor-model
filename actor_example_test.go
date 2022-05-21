package actor

import (
	"context"
	"fmt"
	"testing"
)

// ArithmeticActors A vague interface for actors with arithmetic operations
type ArithmeticActors interface {
	Actor
	Add(int64)
	Sub(int64)
}

// Actor
type CounterActor struct {
	pid PID

	counter int64
}

func NewCounterActor(pid PID) *CounterActor {
	return &CounterActor{pid: pid}
}

func (a *CounterActor) GetPID() PID { return a.pid }

func (a *CounterActor) Receive(ctx *Context, msg any) {
	defer ctx.Finish(nil)

	switch msg := msg.(type) {
	case AdditionMessage:
		a.Add(int64(msg))
	case SubtractionMessage:
		a.Sub(int64(msg))
	case PrintMessage:
		fmt.Printf("CounterActor[%d]: %d\n", a.pid, a.counter)
	default:
		ctx.Finish(ErrUnexpectedMessageType)
		return
	}
}

func (a *CounterActor) Add(num int64) { a.counter += num }

func (a *CounterActor) Sub(num int64) { a.counter -= num }

func (a *CounterActor) Get() int64 { return a.counter }

// Message Types

type AdditionMessage int64

type SubtractionMessage int64

type PrintMessage struct{}

func TestExample(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	supervisor := NewSupervisor(ctx)

	var actor *CounterActor
	pid, err := supervisor.Start(func(pid PID, _ Dispatcher) (Actor, error) {
		actor = NewCounterActor(pid)
		return actor, nil
	})

	if err != nil {
		t.Fatal(err)
	}

	add := AdditionMessage(1)
	err = supervisor.Send(pid, add)
	if err != nil {
		t.Fatal(err)
	}

	add = AdditionMessage(2)
	err = supervisor.Send(pid, add)
	if err != nil {
		t.Fatal(err)
	}

	err = supervisor.Send(pid, PrintMessage{})
	if err != nil {
		t.Fatal(err)
	}

	c := actor.Get()
	if c != 3 {
		t.Fatalf("counter value was %d, expected %d", c, 3)
	}
}
