package scenario

import (
	"errors"
	"fmt"
	"net/http"
)

type clientPool struct {
	// storage for our http.Clients
	clients []chan *http.Client
	factory Factory
	N       int
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

	N := 4

	pool := &clientPool{
		clients: make([]chan *http.Client, N),
		factory: factory,
		N:       N,
	}

	for i := 0; i < N; i++ {
		pool.clients[i] = make(chan *http.Client, maxCap/N)
	}

	// create initial clients, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		client := pool.factory()
		pool.clients[i%N] <- client
	}

	return pool, nil
}

func (c *clientPool) Get() *http.Client {
	var client *http.Client
	// TODO N = 4
	select {
	case client = <-c.clients[0]:
	case client = <-c.clients[1]:
	case client = <-c.clients[2]:
	case client = <-c.clients[3]:
	// case client = <-c.clients[4]:
	// case client = <-c.clients[5]:
	// case client = <-c.clients[6]:
	// case client = <-c.clients[7]:
	// case client = <-c.clients[8]:
	// case client = <-c.clients[9]:

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
	case c.clients[0] <- client:
		return nil
	case c.clients[1] <- client:
		return nil
	case c.clients[2] <- client:
		return nil
	case c.clients[3] <- client:
		return nil
	// case c.clients[4] <- client:
	// 	return nil
	// case c.clients[5] <- client:
	// 	return nil
	// case c.clients[6] <- client:
	// 	return nil
	// case c.clients[7] <- client:
	// 	return nil
	// case c.clients[8] <- client:
	// 	return nil
	// case c.clients[9] <- client:
	// 	return nil

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
	fmt.Println(c.Len())
	for i := 0; i < c.N; i++ {
		close(c.clients[i])
	}
	for _, cp := range c.clients {
		for c := range cp {
			c.CloseIdleConnections()
		}
	}
}
