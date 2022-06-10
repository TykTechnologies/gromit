package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var testApp *App

// setup environment for the test run and cleanup after
func TestMain(m *testing.M) {
	var s *httptest.Server
	s, testApp = StartTestServer("../testdata/env-config.yaml")
	defer s.Close()
	code := m.Run()
	os.Exit(code)
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

// executeMockRequest will make a mock http request
func executeMockRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	testApp.Router.ServeHTTP(rr, req)

	return rr
}

func TestInfraURLs(t *testing.T) {
	// Use this formulation of sub-tests when ordering matters
	// GetLoglvl below works because InfoLvl has set it
	cases := []APITestCase{
		{
			Name:       "Healthcheck",
			Endpoint:   "/healthcheck",
			HTTPStatus: http.StatusOK,
			HTTPMethod: "GET",
		},
		{
			Name:         "InfoLogLvl",
			Endpoint:     "/loglevel/info",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"level":"info"}`,
			HTTPMethod:   "PUT",
		},
		{
			Name:         "GetLoglvl",
			Endpoint:     "/loglevel",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"level":"info"}`,
			HTTPMethod:   "GET",
		},
		{
			Name:         "DebugLogLvl",
			Endpoint:     "/loglevel/debug",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"level":"debug"}`,
			HTTPMethod:   "PUT",
		},
	}
	runSubTests(t, cases)
}

func runSubTests(t *testing.T, cases []APITestCase) {
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.HTTPMethod, tc.Endpoint, strings.NewReader(tc.Payload))
			response := executeMockRequest(req)

			checkResponseCode(t, tc.HTTPStatus, response.Code)

			if tc.ResponseJSON != "" {
				require.JSONEq(t, tc.ResponseJSON, response.Body.String())
				if body := response.Body.String(); body != tc.ResponseJSON {
					t.Errorf("Expected %s. Got %s", tc.ResponseJSON, body)
				}
			}
		})
	}
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
