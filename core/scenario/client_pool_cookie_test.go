package scenario

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.ddosify.com/ddosify/core/types"
)

// If the user agent receives a new cookie with the same cookie-name,
// domain-value, and path-value as a cookie that it has already stored,
// the existing cookie is evicted and replaced with the new cookie

func TestCookieManagerInRepeatedModeOnlySetInFirstIter(t *testing.T) {
	t.Parallel()

	cookieName := "test"

	// cookie value sent by server (first step)
	value1 := "test1"
	value2 := "test2" // login endpoint will set this value to cookie in second iteration, but we expect it to be ignored

	// cookies sent to second step
	var cookieInFirstCall string
	var cookieInSecondCall string

	loginCallCount := 0
	orderCallCount := 0

	firstReqHandler := func(w http.ResponseWriter, r *http.Request) {
		// set cookie, act as server
		var val string
		if loginCallCount == 0 {
			val = value1
		} else {
			val = value2
		}

		cookie := http.Cookie{Name: cookieName, Value: val}
		http.SetCookie(w, &cookie)
		loginCallCount++
	}

	secondReqHandler := func(w http.ResponseWriter, r *http.Request) {
		// check cookie sent by client
		ck, _ := r.Cookie(cookieName)
		if orderCallCount == 0 {
			cookieInFirstCall = ck.Value
		} else {
			cookieInSecondCall = ck.Value
		}
		orderCallCount++
	}

	pathFirst := "/login"
	pathSecond := "/order"

	mux := http.NewServeMux()
	mux.HandleFunc(pathFirst, firstReqHandler)
	mux.HandleFunc(pathSecond, secondReqHandler)

	host := httptest.NewServer(mux)
	defer host.Close()

	// make sure we get the same client in second iteration, 1,1 means we have only one client
	pool, _ := NewClientPool(1, 1, types.EngineModeRepeatedUser, createClientFactoryMethod(types.EngineModeRepeatedUser), defaultClose)

	c := pool.Get()
	// using same client

	// first iteration
	c.Get(host.URL + pathFirst)
	c.Get(host.URL + pathSecond)

	// put client back to pool, so we can reuse it in second iteration
	pool.Put(c)
	c = pool.Get()

	// second iteration
	c.Get(host.URL + pathFirst)
	c.Get(host.URL + pathSecond)

	if cookieInFirstCall != cookieInSecondCall {
		t.Errorf("TestCookieManagerInRepeatedModeOnlySetInFirstIter, cookie should be same in second iteration")
	}
}

func TestSetCookiesAppendToCurrentSliceOfCookies(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	cookie1 := &http.Cookie{Name: "test1", Value: "test1"}
	cookie2 := &http.Cookie{Name: "test2", Value: "test2"}

	// set cookie
	url := url.URL{Scheme: "http", Host: "test.com"}
	jar.SetCookies(&url, []*http.Cookie{cookie1})
	jar.SetCookies(&url, []*http.Cookie{cookie2})

	cookies := jar.Cookies(&url)

	if len(cookies) != 2 {
		t.Errorf("TestCookieSetOverrides, expected 2 cookies, got %d", len(cookies))
	}
}

func TestSetCookiesOverridesCookieWithSameName(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	cookie1 := &http.Cookie{Name: "test", Value: "test1"}
	cookie2 := &http.Cookie{Name: "test", Value: "test2"}

	// set cookie
	url := url.URL{Scheme: "http", Host: "test.com"}
	jar.SetCookies(&url, []*http.Cookie{cookie1})
	jar.SetCookies(&url, []*http.Cookie{cookie2})

	cookies := jar.Cookies(&url)

	if len(cookies) != 1 || cookies[0].Value != "test2" {
		t.Errorf("TestSetCookiesOverridesCookieWithSameName, expected 1 cookie with value 'test2', got %d", len(cookies))
	}
}

func TestSetCookiesDeletesIfUnnecessary(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	cookie1 := &http.Cookie{Name: "test", Value: "test1", Secure: false}
	cookie2 := &http.Cookie{Name: "test", Value: "test2", Secure: true}

	// set cookie
	url := url.URL{Scheme: "http", Host: "test.com"}
	jar.SetCookies(&url, []*http.Cookie{cookie1})
	cookies := jar.Cookies(&url)
	if len(cookies) != 1 {
		t.Errorf("TestSetCookiesDeletesIfUnnecessary, expected 1 cookie with value 'test1', got %d", len(cookies))
	}
	jar.SetCookies(&url, []*http.Cookie{cookie2})
	cookies = jar.Cookies(&url)

	// cookiejar deletes cookies with same name if url scheme is http and cookie is secure
	if len(cookies) != 0 {
		t.Errorf("TestSetCookiesDeletesIfUnnecessary, expected 1 cookie with value 'test2', got %d", len(cookies))
	}
}

func TestSetCookiesUrlScheme(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	cookie1 := &http.Cookie{Name: "test1", Value: "test1"}
	cookie2 := &http.Cookie{Name: "test2", Value: "test2"}

	// set cookie
	url := url.URL{Scheme: "", Host: "test.com"}
	// expect set cookies to be ignored since url scheme is empty
	jar.SetCookies(&url, []*http.Cookie{cookie1})
	jar.SetCookies(&url, []*http.Cookie{cookie2})

	cookies := jar.Cookies(&url)

	if len(cookies) != 0 {
		t.Errorf("TestSetCookiesUrlScheme, expected 0 cookies, got %d", len(cookies))
	}
}

func TestSetCookiesSecure(t *testing.T) {
	t.Parallel()

	secureCookieName := "https-cookie"
	secureCookieVal := "secure-cookie"

	var httpServerGotCookie *http.Cookie
	reqHandler := func(w http.ResponseWriter, r *http.Request) {
		httpServerGotCookie, _ = r.Cookie(secureCookieName)
	}

	var httpsServerGotCookie *http.Cookie
	secureReqHandler := func(w http.ResponseWriter, r *http.Request) {
		httpsServerGotCookie, _ = r.Cookie(secureCookieName)
	}

	path := "/default"
	mux := http.NewServeMux()
	mux.HandleFunc(path, reqHandler)

	host := httptest.NewServer(mux)
	defer host.Close()

	pathSecure := "/secure"
	muxHttps := http.NewServeMux()
	muxHttps.HandleFunc(pathSecure, secureReqHandler)

	secureHost := httptest.NewTLSServer(muxHttps)
	defer secureHost.Close()

	c := secureHost.Client()
	c.Jar, _ = cookiejar.New(nil)

	secureCookie := http.Cookie{Name: secureCookieName, Value: secureCookieVal, Secure: true}
	// set cookies
	url, _ := url.Parse(secureHost.URL)
	c.Jar.SetCookies(url, []*http.Cookie{&secureCookie})

	c.Get(host.URL + path)
	c.Get(secureHost.URL + pathSecure)

	// expect secure cookie to be sent only to secure host

	if httpServerGotCookie != nil {
		t.Errorf("TestSetCookiesSecure, expected no cookie to be sent to http host, got %s", httpServerGotCookie.Value)
	}

	if httpsServerGotCookie == nil || httpsServerGotCookie.Value != secureCookieVal {
		t.Errorf("TestSetCookiesSecure, expected cookie to be sent to https host, got %s", httpsServerGotCookie.Value)
	}
}

func TestPutInitialCookiesInJarFactory(t *testing.T) {
	mode := types.EngineModeDistinctUser
	pool, _ := NewClientPool(1, 1, mode, putInitialCookiesInJarFactory(mode,
		[]*http.Cookie{
			{
				Name:   "test",
				Value:  "test",
				Domain: "ddosify.com",
				Secure: false,
			},
			{
				Name:   "test",
				Value:  "test",
				Domain: "servdown.com",
				Secure: true,
			}}), defaultClose)

	c := pool.Get()

	cookies := c.Jar.Cookies(&url.URL{Scheme: "http", Host: "ddosify.com"})

	if len(cookies) != 1 {
		t.Errorf("TestPutInitialCookiesInJarFactory, expected 1 cookie, got %d", len(cookies))
	}

	if cookies[0].Value != "test" {
		t.Errorf("TestPutInitialCookiesInJarFactory, expected cookie value 'test', got %s", cookies[0].Value)
	}

	cookies = c.Jar.Cookies(&url.URL{Scheme: "https", Host: "servdown.com"})

	if len(cookies) != 1 {
		t.Errorf("TestPutInitialCookiesInJarFactory, expected 1 cookie, got %d", len(cookies))
	}

	if cookies[0].Value != "test" {
		t.Errorf("TestPutInitialCookiesInJarFactory, expected cookie value 'test', got %s", cookies[0].Value)
	}
}
