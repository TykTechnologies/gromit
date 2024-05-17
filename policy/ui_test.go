package policy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testConfig = "../testdata/test-variations.yaml"

// executeRequest, creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
func executeRequest(req *http.Request, s *Server) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.Router.ServeHTTP(rr, req)

	return rr
}

// checkResponseCode is a simple utility to check the response code
// of the response
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestHelloWorld(t *testing.T) {
	s := CreateNewServer(testConfig)
	s.MountHandlers()
	req, _ := http.NewRequest("GET", "/", nil)

	response := executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	assert.Equal(t, "Hello World!", response.Body.String())
}
