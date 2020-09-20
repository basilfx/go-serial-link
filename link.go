package link

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/basilfx/go-utilities/taskrunner"
	"github.com/twinj/uuid"
)

// Listener represets the identifier of a listener.
type Listener uuid.UUID

// RequestTimeout is the time the Request method waits for a response.
const RequestTimeout = 5 * time.Second

// WriterChannelSize is the size of the writer channel.
const WriterChannelSize = 32

// ListenerChannelSize is the size of the channel that is created for each
// listener.
const ListenerChannelSize = 32

// Link represents a link.
type Link struct {
	stream io.ReadWriter

	taskRunner *taskrunner.TaskRunner

	writer    chan Message
	listeners map[Listener]chan Message
	lock      sync.RWMutex

	counter uint32
}

// New returns a new instance initilized instance of Link.
func New() *Link {
	return &Link{
		writer:     make(chan Message, WriterChannelSize),
		listeners:  map[Listener]chan Message{},
		taskRunner: taskrunner.New(),
	}
}

// Register interest in received messages.
func (l *Link) Register() (Listener, chan Message) {
	c := make(chan Message, ListenerChannelSize)
	id := Listener(uuid.NewV4())

	l.lock.Lock()
	defer l.lock.Unlock()

	l.listeners[id] = c

	return id, c
}

// Unregister interest in received messages.
func (l *Link) Unregister(id Listener) {
	l.lock.Lock()
	defer l.lock.Unlock()

	c, ok := l.listeners[id]

	if !ok {
		return
	}

	delete(l.listeners, id)

	close(c)
}

// Request a message, and wait for a response or a time-out.
func (l *Link) Request(message Message) (Message, error) {
	id, c := l.Register()
	defer l.Unregister(id)

	// Create goroutine that waits for a proper response.
	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	response := make(chan Message)
	defer close(response)

	message.Type = MessageRequest
	message.ID = atomic.AddUint32(&l.counter, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-c:
				if r.ID == message.ID {
					response <- r
					return
				}
			}
		}
	}()

	// Send the request message to the writer.
	err := l.Write(message)

	if err != nil {
		return Message{}, err
	}

	// Wait for the response.
	select {
	case <-ctx.Done():
		return Message{}, errors.New("timeout while waiting for response")
	case r := <-response:
		return r, nil
	}
}

// Reply to a given request message.
func (l *Link) Reply(request Message, message Message) error {
	if request.Type != MessageRequest {
		return errors.New("not a request")
	}

	message.Type = MessageResponse
	message.ID = request.ID

	return l.Write(message)
}

// Notify will send a notification message.
func (l *Link) Notify(message Message) error {
	message.Type = MessageNotification
	message.ID = 0

	return l.Write(message)
}

// Write a message to the link.
func (l *Link) Write(message Message) error {
	select {
	case l.writer <- message:
		return nil
	default:
		return errors.New("writer channel full")
	}
}

// Serve a link.
func (l *Link) Serve(stream io.ReadWriter) {
	l.stream = stream

	l.taskRunner.RunWithCancel("Link.Writer", l.writerTask)
	l.taskRunner.RunWithCancel("Link.Reader", l.readerTask)

	// Wait for both goroutines to complete.
	l.taskRunner.Wait()
}

// Shutdown the link. This does not close the underlying stream.
func (l *Link) Shutdown() {
	if l.taskRunner != nil {
		l.taskRunner.Cancel()
	}
}
