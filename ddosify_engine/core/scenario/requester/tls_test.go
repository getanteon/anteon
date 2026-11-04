package requester

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.ddosify.com/ddosify/v2/core/types"
)

func TestCertificatesVerifiedUnlessInsecure(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	res := sendStep(t, types.ScenarioStep{ID: 1, Method: http.MethodGet, URL: srv.URL, Timeout: 5})
	if res.Err.Type == "" {
		t.Error("a self-signed certificate must be refused by default")
	}

	res = sendStep(t, types.ScenarioStep{ID: 1, Method: http.MethodGet, URL: srv.URL, Timeout: 5, Insecure: true})
	if res.Err.Type != "" || res.StatusCode != 200 {
		t.Errorf("insecure opts out of verification, got %v %d", res.Err, res.StatusCode)
	}
}
