package scenario

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"go.ddosify.com/ddosify/core/types"
	"go.ddosify.com/ddosify/core/util"
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
func NewClientPool(initialCap, maxCap int, engineMode string, factory ClientFactoryMethod, close ClientCloseMethod) (*util.Pool[*http.Client], error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	pool := &util.Pool[*http.Client]{
		Items:   make(chan *http.Client, maxCap),
		Factory: factory,
		Close:   close,
		AfterPut: func(client *http.Client) {
			// if engine is in repeated mode, notify jar that cookies are already set
			// to avoid setting them again in the next iteration
			if engineMode == types.EngineModeRepeatedUser && client.Jar != nil && !client.Jar.(*cookieJarRepeated).firstIterPassed {
				client.Jar.(*cookieJarRepeated).firstIterPassed = true
			}
		},
	}

	// create initial clients, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		client := pool.Factory()
		pool.Items <- client
	}

	return pool, nil
}

type cookieJarRepeated struct {
	defaultCookieJar *cookiejar.Jar
	firstIterPassed  bool
}

func NewCookieJarRepeated() (*cookieJarRepeated, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &cookieJarRepeated{defaultCookieJar: jar}, nil
}

// SetCookies implements the http.CookieJar interface.
// Only set cookies if they are not already set for repeated mode.
func (c *cookieJarRepeated) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if !c.firstIterPassed {
		// execute default behavior if no cookies are set
		c.defaultCookieJar.SetCookies(u, cookies)
	}
}

// Cookies implements the http.CookieJar interface.
func (c *cookieJarRepeated) Cookies(u *url.URL) []*http.Cookie {
	return c.defaultCookieJar.Cookies(u)
}

var defaultFactory = func() *http.Client {
	return &http.Client{}
}

var defaultClose = func(c *http.Client) {
	c.CloseIdleConnections()
}

// createFactoryMethod returns a Factory function based on the engine mode.
func createClientFactoryMethod(mode string, opts ...func(http.CookieJar)) ClientFactoryMethod {
	if mode == types.EngineModeRepeatedUser {
		return func() *http.Client {
			jar, err := NewCookieJarRepeated()
			if err != nil {
				return defaultFactory() // no cookie jar, use default factory
			}

			for _, opt := range opts {
				opt(jar)
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

		for _, opt := range opts {
			opt(jar)
		}
		return &http.Client{Jar: jar}
	}
}
