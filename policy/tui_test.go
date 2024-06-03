package policy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConfig = "../testdata/test-variations.yaml"

// executeMockRequest, creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
func executeMockRequest(req *http.Request, s *Server) *httptest.ResponseRecorder {
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

func TestPing(t *testing.T) {
	s := CreateNewServer(testConfig)
	req, _ := http.NewRequest("GET", "/ping", nil)

	response := executeMockRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	assert.Equal(t, "Pong!", response.Body.String())
}

// APITestCases are for testcases that exercise the rest API
type APITestCase struct {
	Name          string
	Endpoint      string
	Payload       string
	HTTPStatus    int
	ResponseJSON  string
	HTTPMethod    string
	RequestParams string
}

func TestVariations(t *testing.T) {
	// Order matters, delete after creating
	cases := []APITestCase{
		{
			Name:         "EnvFiles",
			Endpoint:     "/api/repo1/br0/tr0/ts0/EnvFiles",
			ResponseJSON: `[{"cache":"repo1-redis0", "config":"repo1-conf0", "db":"repo1-db0"}]`,
			HTTPStatus:   http.StatusOK,
			HTTPMethod:   "GET",
		},
		{
			Name:         "Pump",
			Endpoint:     "/api/repo0/br1/tr1/ts0/Pump",
			ResponseJSON: `["pump-br1", "master"]`,
			HTTPStatus:   http.StatusOK,
			HTTPMethod:   "GET",
		},
		{
			Name:         "Sink",
			Endpoint:     "/api/repo1/br0/tr1/ts0/Sink",
			ResponseJSON: `["sink-br0", "master"]`,
			HTTPStatus:   http.StatusOK,
			HTTPMethod:   "GET",
		},
	}
	runSubTests(t, cases)
}

func runSubTests(t *testing.T, cases []APITestCase) {
	s := CreateNewServer(testConfig)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.HTTPMethod, tc.Endpoint, strings.NewReader(tc.Payload))
			response := executeMockRequest(req, s)

			checkResponseCode(t, tc.HTTPStatus, response.Code)

			if tc.ResponseJSON != "" {
				require.JSONEq(t, tc.ResponseJSON, response.Body.String())
			}
		})
	}
}
