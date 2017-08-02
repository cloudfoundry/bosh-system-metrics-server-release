package tokenchecker_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"net/http/httputil"
	"sync"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/tokenchecker"
	"encoding/base64"
	"fmt"
)

func TestCheckToken(t *testing.T) {
	RegisterTestingT(t)

	sas := newSpyAuthServer(200)
	testAuthServer := httptest.NewServer(sas)
	defer testAuthServer.Close()

	client := tokenchecker.New(&tokenchecker.TokenCheckerConfig{
		UaaURL:      testAuthServer.URL,
		UaaClient:   "test-client",
		UaaPassword: "test-secret",
		Authority:   "bosh.system_metrics.read",
	})

	err := client.CheckToken("fake-token")

	Expect(err).ToNot(HaveOccurred())
	receivedRequest := sas.lastRequest
	Expect(receivedRequest.ParseForm()).ToNot(HaveOccurred())

	Expect(receivedRequest.Method).To(Equal(http.MethodPost))
	Expect(receivedRequest.URL.Path).To(Equal("/check_token"))
	Expect(receivedRequest.Form.Get("token")).To(Equal("fake-token"))
	Expect(receivedRequest.Form.Get("scopes")).To(Equal("bosh.system_metrics.read"))

	expectedAuthorization := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", "test-client", "test-secret")))
	Expect(receivedRequest.Header.Get("Authorization")).To(Equal(fmt.Sprintf("Basic %s", expectedAuthorization)))
}

func TestCheckToken_withBadUAAUrl(t *testing.T) {
	RegisterTestingT(t)

	client := tokenchecker.New(&tokenchecker.TokenCheckerConfig{
		UaaURL:      "://bad",
		UaaClient:   "test-client",
		UaaPassword: "test-secret",
		Authority:   "bosh.system_metrics.read",
	})

	err := client.CheckToken("fake-token")

	Expect(err).To(HaveOccurred())
}

func TestCheckToken_withBadUAAResponse(t *testing.T) {
	RegisterTestingT(t)

	sas := newSpyAuthServer(404)
	testAuthServer := httptest.NewServer(sas)
	defer testAuthServer.Close()

	client := tokenchecker.New(&tokenchecker.TokenCheckerConfig{
		UaaURL:      testAuthServer.URL,
		UaaClient:   "test-client",
		UaaPassword: "test-secret",
		Authority:   "bosh.system_metrics.read",
	})

	err := client.CheckToken("fake-token")

	Expect(err).To(HaveOccurred())
}

func TestCheckToken_whenUAAIsDown(t *testing.T) {
	RegisterTestingT(t)

	client := tokenchecker.New(&tokenchecker.TokenCheckerConfig{
		UaaURL:      "http://localhost:343343",
		UaaClient:   "test-client",
		UaaPassword: "test-secret",
		Authority:   "bosh.system_metrics.read",
	})

	err := client.CheckToken("fake-token")

	Expect(err).To(HaveOccurred())
}

type spyAuthServer struct {
	mu          sync.Mutex
	lastRequest *http.Request
	status      int

}

func newSpyAuthServer(status int) *spyAuthServer {
	return &spyAuthServer{
		status: status,
	}
}

func (a *spyAuthServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	httputil.DumpRequest(r, true)
	a.lastRequest = r

	w.WriteHeader(a.status)
}
