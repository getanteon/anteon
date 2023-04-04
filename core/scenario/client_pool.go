package scenario

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"go.ddosify.com/ddosify/core/types"
)

type clientPool struct {
	// storage for our http.Clients
	clients    chan *http.Client
	factory    Factory
	engineMode string
}

// Factory is a function to create new connections.
type Factory func() *http.Client

// NewClientPool returns a new pool based on buffered channels with an initial
// capacity and maximum capacity. Factory is used when initial capacity is
// greater than zero to fill the pool. A zero initialCap doesn't fill the Pool
// until a new Get() is called. During a Get(), If there is no new client
// available in the pool, a new client will be created via the Factory()
// method.
func NewClientPool(initialCap, maxCap int, engineMode string, factory Factory) (*clientPool, error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	pool := &clientPool{
		clients:    make(chan *http.Client, maxCap),
		factory:    factory,
		engineMode: engineMode,
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
		// if engine is in repeated mode, notify jar that cookies are already set
		// to avoid setting them again in the next iteration
		if c.engineMode == types.EngineModeRepeatedUser && client.Jar != nil && !client.Jar.(*cookieJarRepeated).set {
			client.Jar.(*cookieJarRepeated).set = true
		}
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

type cookieJarRepeated struct {
	defaultCookieJar *cookiejar.Jar
	set              bool
}

func NewCoooieJarRepeated() (*cookieJarRepeated, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &cookieJarRepeated{defaultCookieJar: jar}, nil
}

// SetCookies implements the http.CookieJar interface.
// Only set cookies if they are not already set for repeated mode.
func (c *cookieJarRepeated) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if !c.set {
		// execute default behavior if no cookies are set
		c.defaultCookieJar.SetCookies(u, cookies)
		c.set = true
	}
}

// Cookies implements the http.CookieJar interface.
func (c *cookieJarRepeated) Cookies(u *url.URL) []*http.Cookie {
	return c.defaultCookieJar.Cookies(u)
}

var defaultFactory = func() *http.Client {
	return &http.Client{}
}

func createFactoryMethod(mode string) Factory {
	if mode == types.EngineModeRepeatedUser {
		return func() *http.Client {
			jar, err := NewCoooieJarRepeated()
			if err != nil {
				return defaultFactory() // no cookie jar, use default factory
			}
			return &http.Client{Jar: jar}
		}
	}

	// distinct users mode
	return func() *http.Client {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return defaultFactory() // no cookie jar, use default factory
		}
		return &http.Client{Jar: jar}
	}
}
