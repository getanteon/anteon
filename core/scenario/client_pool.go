package scenario

import (
	"errors"
	"net/http"
)

type clientPool struct {
	// storage for our http.Clients
	clients chan *http.Client
	factory Factory
}

// Factory is a function to create new connections.
type Factory func() *http.Client

// NewClientPool returns a new pool based on buffered channels with an initial
// capacity and maximum capacity. Factory is used when initial capacity is
// greater than zero to fill the pool. A zero initialCap doesn't fill the Pool
// until a new Get() is called. During a Get(), If there is no new client
// available in the pool, a new client will be created via the Factory()
// method.
func NewClientPool(initialCap, maxCap int, factory Factory) (*clientPool, error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	pool := &clientPool{
		clients: make(chan *http.Client, maxCap),
		factory: factory,
	}

	// create initial clients, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		client := pool.factory()
		pool.clients <- client
	}

	return pool, nil
}

func (c *clientPool) Get() *http.Client {
	var client *http.Client
	select {
	case client = <-c.clients:
	default:
		client = c.factory()
	}
	return client
}

func (c *clientPool) Put(client *http.Client) error {
	if client == nil {
		return errors.New("client is nil. rejecting")
	}

	if c.clients == nil {
		// pool is closed, close passed client
		client.CloseIdleConnections()
		return nil
	}

	// put the resource back into the pool. If the pool is full, this will
	// block and the default case will be executed.
	select {
	case c.clients <- client:
		return nil
	default:
		// pool is full, close passed client
		client.CloseIdleConnections()
		return nil
	}
}

func (c *clientPool) Len() int {
	return len(c.clients)
}

func (c *clientPool) Done() {
	close(c.clients)
	for c := range c.clients {
		c.CloseIdleConnections()
	}
}
