package scenario

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.ddosify.com/ddosify/core/types"
)

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

	pool, _ := NewClientPool(1, 1, types.EngineModeRepeatedUser, createFactoryMethod(types.EngineModeRepeatedUser))

	c := pool.Get()
	// using same client

	// first iteration
	c.Get(host.URL + pathFirst)
	c.Get(host.URL + pathSecond)

	// second iteration
	c.Get(host.URL + pathFirst)
	c.Get(host.URL + pathSecond)

	if cookieInFirstCall != cookieInSecondCall {
		t.Errorf("TestCookieManagerInRepeatedModeOnlySetInFirstIter, cookie should be same in second iteration")
	}
}
