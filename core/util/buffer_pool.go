package util

import (
	"bytes"
	"errors"
)

// Factory is a function to create new connections.
type BufferFactoryMethod func() *bytes.Buffer
type BufferCloseMethod func(*bytes.Buffer)

func NewBufferPool(initialCap, maxCap int, factory BufferFactoryMethod, close BufferCloseMethod) (*Pool[*bytes.Buffer], error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	pool := &Pool[*bytes.Buffer]{
		Items:   make(chan *bytes.Buffer, maxCap),
		Factory: factory,
		Close:   close,
	}

	// create initial clients, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		client := pool.Factory()
		pool.Items <- client
	}

	return pool, nil
}
