package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var a App

// setup environment for the test run and cleanup after
func TestMain(m *testing.M) {
	a.Init("../testdata/ca.pem")

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
	a.Router.ServeHTTP(rr, req)

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

func TestPositives(t *testing.T) {
	// Order matters, delete after creating
	cases := []APITestCase{
		{
			Name:         "InsertTestEnv",
			Endpoint:     "/env/test",
			Payload:      `{"tyk":"sha1","tyk-analytics":"sha2","tyk-pump":"sha3"}`,
			ResponseJSON: `{"name":"test","state":"new","tyk":"sha1","tyk-analytics":"sha2","tyk-pump":"sha3"}`,
			HTTPStatus:   http.StatusCreated,
			HTTPMethod:   "PUT",
		},
		{
			Name:         "GetTestEnv",
			Endpoint:     "/env/test",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"test","state":"new","tyk":"sha1","tyk-analytics":"sha2","tyk-pump":"sha3"}`,
			HTTPMethod:   "GET",
		},
		{
			Name:         "UpdateTestEnv",
			Endpoint:     "/env/test",
			Payload:      `{"tyk":"updated-sha"}`,
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"test","state":"new","tyk":"updated-sha","tyk-analytics":"sha2","tyk-pump":"sha3"}`,
			HTTPMethod:   "PATCH",
		},
		{
			Name:         "CheckUpdate",
			Endpoint:     "/env/test",
			Payload:      `{"name":"test", "tyk":"updated-sha"}`,
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"test","state":"new","tyk":"updated-sha","tyk-analytics":"sha2","tyk-pump":"sha3"}`,
			HTTPMethod:   "GET",
		},
		{
			Name:       "Delete",
			Endpoint:   "/env/test",
			HTTPStatus: http.StatusAccepted,
			HTTPMethod: "DELETE",
		},
	}
	runSubTests(t, cases)
}

func TestNegatives(t *testing.T) {
	// Order matters, delete after creating
	cases := []APITestCase{
		{
			Name:       "InsertTestEnv",
			Endpoint:   "/env/test-neg",
			Payload:    `{"name":"test", "tyk":"sha1", "tyk-analytics":"sha2", "tyk-pump":"sha3"}`,
			HTTPStatus: http.StatusCreated,
			HTTPMethod: "PUT",
		},
		{
			Name:       "Duplicate",
			Endpoint:   "/env/test-neg",
			Payload:    `{"name":"test", "tyk":"sha1", "tyk-analytics":"sha2", "tyk-pump":"sha3"}`,
			HTTPStatus: http.StatusCreated,
			HTTPMethod: "PUT",
		},
		{
			Name:       "Delete",
			Endpoint:   "/env/test-neg",
			HTTPStatus: http.StatusAccepted,
			HTTPMethod: "DELETE",
		},
		{
			Name:       "DeleteUnknown",
			Endpoint:   "/env/unknown",
			HTTPStatus: http.StatusNotFound,
			HTTPMethod: "DELETE",
		},
		{
			Name:         "UnknownEnv",
			Endpoint:     "/env/unknown-env",
			HTTPStatus:   http.StatusNotFound,
			ResponseJSON: `{"error":"could not find env unknown-env"}`,
			HTTPMethod:   "GET",
		},
	}
	runSubTests(t, cases)
}

func TestNewBuild(t *testing.T) {
	cases := []APITestCase{
		{
			Name:       "PlainPump",
			Endpoint:   "/newbuild",
			HTTPStatus: http.StatusOK,
			Payload:    `{"repo":"tyk-pump","ref":"app-test","sha":"sha-pump"}`,
			HTTPMethod: "POST",
		},
		{
			Name:         "CheckPlainPump",
			Endpoint:     "/env/app-test",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"app-test","state":"new","tyk":"master","tyk-analytics":"master","tyk-pump":"sha-pump"}`,
			HTTPMethod:   "GET",
		},
		{
			Name:       "BranchWithDots",
			Endpoint:   "/newbuild",
			HTTPStatus: http.StatusOK,
			Payload:    `{"repo":"tyk-pump","ref":"release-3.1.0","sha":"sha-pump"}`,
			HTTPMethod: "POST",
		},
		{
			Name:         "CheckBranchWithDots",
			Endpoint:     "/env/release-310",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"release-310","state":"new","tyk":"master","tyk-analytics":"master","tyk-pump":"sha-pump"}`,
			HTTPMethod:   "GET",
		},
		{
			Name:       "GHStyleGateway",
			Endpoint:   "/newbuild",
			HTTPStatus: http.StatusOK,
			Payload:    `{"repo":"TykTechnologies/tyk","ref":"refs/heads/integration/app-test","sha":"sha-gw"}`,
			HTTPMethod: "POST",
		},
		{
			Name:         "CheckGHStyleGateway",
			Endpoint:     "/env/app-test",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"app-test","state":"new","tyk":"sha-gw","tyk-analytics":"master","tyk-pump":"master"}`,
			HTTPMethod:   "GET",
		},
		{
			Name:       "URLEncGHStyleDashboard",
			Endpoint:   "/newbuild",
			HTTPStatus: http.StatusOK,
			Payload:    `{"repo":"TykTechnologies%2Ftyk-analytics","ref":"refs%2Fheads%2Fintegration%2Fapp-test","sha":"sha-db"}`,
			HTTPMethod: "POST",
		},
		{
			Name:         "CheckURLEncStyleDashboard",
			Endpoint:     "/env/app-test",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"app-test","state":"new","tyk":"master","tyk-analytics":"sha-db","tyk-pump":"master"}`,
			HTTPMethod:   "GET",
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
