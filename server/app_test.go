package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/stretchr/testify/require"
)

var a App

func TestMain(m *testing.M) {
	os.Setenv("GROMIT_TABLENAME", "GromitTest")
	os.Setenv("GROMIT_REPOS", "tyk,tyk-analytics,tyk-pump")
	os.Setenv("GROMIT_REGISTRYID", "754489498669")
	a.Init("../ccerts/ca.pem")
	code := m.Run()
	devenv.DeleteTable(a.DB, "GromitTest")
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
			Name:       "InsertTestEnv",
			Endpoint:   "/env/test",
			Payload:    `{"state":"new","tyk":"sha1","tyk-analytics":"sha2","tyk-pump":"sha3"}`,
			HTTPStatus: http.StatusCreated,
			HTTPMethod: "PUT",
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
			ResponseJSON: `{"state":"new","tyk":"updated-sha"}`,
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
			Endpoint:   "/env/test",
			Payload:    `{"name":"test", "tyk":"sha1", "tyk-analytics":"sha2", "tyk-pump":"sha3"}`,
			HTTPStatus: http.StatusCreated,
			HTTPMethod: "PUT",
		},
		{
			Name:       "Duplicate",
			Endpoint:   "/env/test",
			Payload:    `{"name":"test", "tyk":"sha1", "tyk-analytics":"sha2", "tyk-pump":"sha3"}`,
			HTTPStatus: http.StatusConflict,
			HTTPMethod: "PUT",
		},
		{
			Name:       "Delete",
			Endpoint:   "/env/test",
			HTTPStatus: http.StatusAccepted,
			HTTPMethod: "DELETE",
		},
		{
			Name:       "DeleteUnknown",
			Endpoint:   "/env/unknown",
			HTTPStatus: http.StatusAccepted,
			HTTPMethod: "DELETE",
		},
		{
			Name:         "UnknownEnv",
			Endpoint:     "/env/unknown-env",
			HTTPStatus:   http.StatusNotFound,
			ResponseJSON: `{"error":"does not exist: unknown-env"}`,
			HTTPMethod:   "GET",
		},
	}
	runSubTests(t, cases)
}

func TestNewBuild(t *testing.T) {
	// Use this formulation of sub-tests when ordering matters
	// GetLoglvl below works because InfoLvl has set it
	cases := []APITestCase{
		{
			Name:       "PlainPump",
			Endpoint:   "/newbuild",
			HTTPStatus: http.StatusOK,
			Payload:    `{"repo":"tyk-pump","ref":"test","sha":"sha-pump"}`,
			HTTPMethod: "POST",
		},
		{
			Name:         "CheckPlainPump",
			Endpoint:     "/env/test",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"test","state":"new","tyk":"master","tyk-analytics":"master","tyk-pump":"sha-pump"}`,
			HTTPMethod:   "GET",
		},
		{
			Name:       "GHStyleGateway",
			Endpoint:   "/newbuild",
			HTTPStatus: http.StatusOK,
			Payload:    `{"repo":"TykTechnologies/tyk","ref":"refs/heads/integration/test","sha":"sha-gw"}`,
			HTTPMethod: "POST",
		},
		{
			Name:         "CheckGHStyleGateway",
			Endpoint:     "/env/test",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"test","state":"new","tyk":"sha-gw","tyk-analytics":"master","tyk-pump":"master"}`,
			HTTPMethod:   "GET",
		},
		{
			Name:       "URLEncGHStyleDashboard",
			Endpoint:   "/newbuild",
			HTTPStatus: http.StatusOK,
			Payload:    `{"repo":"TykTechnologies%2Ftyk-analytics","ref":"refs%2Fheads%2Fintegration%2Ftest","sha":"sha-db"}`,
			HTTPMethod: "POST",
		},
		{
			Name:         "CheckURLEncStyleDashboard",
			Endpoint:     "/env/test",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"test","state":"new","tyk":"master","tyk-analytics":"sha-db","tyk-pump":"master"}`,
			HTTPMethod:   "GET",
		},
	}
	runSubTests(t, cases)
}

func runSubTests(t *testing.T, cases []APITestCase) {
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.HTTPMethod, tc.Endpoint, strings.NewReader(tc.Payload))
			response := executeRequest(req)

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

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
