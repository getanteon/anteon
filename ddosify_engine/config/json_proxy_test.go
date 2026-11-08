package config

import (
	"strings"
	"testing"
)

func hammerFromJSON(t *testing.T, cfg string) error {
	t.Helper()
	r, err := NewConfigReader([]byte(cfg), ConfigTypeJson)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.CreateHammer()
	return err
}

func TestCreateHammerProxyFromOsEnv(t *testing.T) {
	t.Setenv("DDOSIFY_TEST_PROXY_URL", "http://10.77.1.4:13128")
	t.Setenv("DDOSIFY_TEST_PROXY_PORT", "3128")
	for cfg, want := range map[string]string{
		`{"proxy": "{{$DDOSIFY_TEST_PROXY_URL}}"}`:                              "http://10.77.1.4:13128",
		`{"proxy": "http://user:pass@proxy_host:{{$DDOSIFY_TEST_PROXY_PORT}}"}`: "http://user:pass@proxy_host:3128",
		`{"proxy": "http://proxy_host:80"}`:                                     "http://proxy_host:80",
	} {
		r, err := NewConfigReader([]byte(cfg), ConfigTypeJson)
		if err != nil {
			t.Fatal(err)
		}
		h, err := r.CreateHammer()
		if err != nil {
			t.Fatalf("%s: %v", cfg, err)
		}
		if h.Proxy.Addr == nil || h.Proxy.Addr.String() != want {
			t.Errorf("%s: proxy %v, want %s", cfg, h.Proxy.Addr, want)
		}
		if h.Proxy.Addr != nil && h.Proxy.Addr.Host == "" {
			t.Errorf("%s: proxy has no host", cfg)
		}
	}
}

func TestCreateHammerProxyMissingOsEnv(t *testing.T) {
	t.Setenv("DDOSIFY_TEST_PROXY_SET", "proxy_host")
	err := hammerFromJSON(t, `{"proxy": "http://{{$DDOSIFY_TEST_PROXY_SET}}:{{$DDOSIFY_TEST_PROXY_UNSET_A}}/{{$DDOSIFY_TEST_PROXY_UNSET_B}}"}`)
	if err == nil {
		t.Fatal("a proxy that needs an unset variable must fail")
	}
	for _, want := range []string{"proxy", "DDOSIFY_TEST_PROXY_UNSET_A", "DDOSIFY_TEST_PROXY_UNSET_B", "not set"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "DDOSIFY_TEST_PROXY_SET ") || strings.Contains(err.Error(), "DDOSIFY_TEST_PROXY_SET,") {
		t.Errorf("a set variable is named: %v", err)
	}
}
