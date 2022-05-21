package actor

import (
	"context"
	"errors"
)

// ErrUnexpectedMessageType error is returned when an actor recieves a message of
// and unexpected type.
var ErrUnexpectedMessageType = errors.New("unexpected message type")

// Context An Actor specific context
type Context struct {
	Actor Actor

	ctx    context.Context
	cancel context.CancelFunc
	err    error
}

// Create a context
func NewContext(ctx context.Context, cancel context.CancelFunc, actor Actor) *Context {
	return &Context{
		Actor:  actor,
		ctx:    ctx,
		cancel: cancel,
		err:    nil,
	}
}

// Done returns a channel that's closed when work is done
func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Finish completes the context, optionally with an error
func (c *Context) Finish(err error) {
	if err != nil {
		c.err = err
	}
	c.cancel()
}

// Err returns any error that occurred
func (c *Context) Err() error {
	return c.err
}

// Actor used to implement an Actor
type Actor interface {
	GetPID() PID
	Receive(*Context, any)
}
