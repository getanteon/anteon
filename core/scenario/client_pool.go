package scenario

import (
	"errors"
	"net/http"
)

// Factory is a function to create new connections.
type ClientFactoryMethod func() *http.Client
type ClientCloseMethod func(*http.Client)

// NewClientPool returns a new pool based on buffered channels with an initial
// capacity and maximum capacity. Factory is used when initial capacity is
// greater than zero to fill the pool. A zero initialCap doesn't fill the Pool
// until a new Get() is called. During a Get(), If there is no new client
// available in the pool, a new client will be created via the Factory()
// method.
func NewClientPool(initialCap, maxCap int, factory ClientFactoryMethod, close ClientCloseMethod) (*Pool[*http.Client], error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	pool := &Pool[*http.Client]{
		items:   make(chan *http.Client, maxCap),
		factory: factory,
		close:   close,
	}

	// create initial clients, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		client := pool.factory()
		pool.items <- client
	}

	return pool, nil
}
