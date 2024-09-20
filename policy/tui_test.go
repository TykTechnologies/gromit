package policy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testConfig = "../testdata/tui"

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

func TestSPA(t *testing.T) {
	creds := getCredentials(`{"user": "pass"}`)
	s := CreateNewServer(testConfig, creds)
	req, _ := http.NewRequest("GET", "/", strings.NewReader(""))
	response := executeMockRequest(req, s)
	if response.Body.Len() < 5000 {
		t.Logf("index.html <5k: %s", response.Body.String())
		t.Fail()
	}
}

// APITestCases are for testcases that exercise the rest API
type APITestCase struct {
	Name          string
	Endpoint      string
	Payload       string
	HTTPStatus    int
	ResponseJSON  string
	ResponseText  string
	HTTPMethod    string
	RequestParams string
}

func TestV1Variations(t *testing.T) {
	// Order matters, delete after creating
	cases := []APITestCase{
		{
			Name:         "EnvFiles",
			Endpoint:     "/api/repo1/br0/tr0/ts0/EnvFiles",
			ResponseJSON: `[{"cache":"redis0", "config":"conf0", "db":"db0", "apimarkers":"am0", "uimarkers":"um0", "gwdash":"branch0"}]`,
			HTTPStatus:   http.StatusOK,
			HTTPMethod:   "GET",
		},
		{
			Name:         "Pump",
			Endpoint:     "/api/repo0/br1/tr1/ts1/Pump",
			ResponseJSON: `["pump-br1", "master"]`,
			HTTPStatus:   http.StatusOK,
			HTTPMethod:   "GET",
		},
		{
			Name:         "Sink",
			Endpoint:     "/api/repo0/unknown/tr0/ts0/Sink",
			ResponseJSON: `["master"]`,
			HTTPStatus:   http.StatusOK,
			HTTPMethod:   "GET",
		},
	}
	runSubTests(t, cases)
}

func TestV2Variations(t *testing.T) {
	// Order matters, delete after creating
	cases := []APITestCase{
		{
			Name:         "EnvFiles",
			Endpoint:     "/v2/prod-variations/repo0/br0/tr0/ts0/EnvFiles.json",
			ResponseJSON: `[{"cache":"redis0", "config":"conf0", "db":"db0", "apimarkers":"am0", "gwdash":"branch0", "uimarkers":"um0"}]`,
			HTTPStatus:   http.StatusOK,
			HTTPMethod:   "GET",
		},
		{
			Name:     "gho",
			Endpoint: "/v2/prod-var/repo0/br1/tr1/ts1.gho",
			ResponseText: `envfiles<<EOF
[{"cache":"redis-br1","db":"db-br1","config":"conf-br1","apimarkers":"br1-am1","uimarkers":"br1-um1","gwdash":""},{"cache":"redis1","db":"db1","config":"conf1","apimarkers":"am1","uimarkers":"um1","gwdash":"branch1"},{"cache":"redis0","db":"db0","config":"conf0","apimarkers":"am0","uimarkers":"um0","gwdash":"branch0"}]
EOF
pump<<EOF
["pump-br1","master"]
EOF
sink<<EOF
["sink-br1","master"]
EOF
distros<<EOF
{"deb":["d1"],"rpm":["d0"]}
EOF
`,
			HTTPStatus: http.StatusOK,
			HTTPMethod: "GET",
		},
		{
			Name:     "field-gho",
			Endpoint: "/v2/prod-variations.yml/repo1/unknown/tr0/ts0/Distros.gho",
			ResponseText: `deb<<EOF
["d1"]
EOF
rpm<<EOF
["d0"]
EOF
`,
			HTTPStatus: http.StatusOK,
			HTTPMethod: "GET",
		},
	}
	runSubTests(t, cases)
}

func runSubTests(t *testing.T, cases []APITestCase) {
	creds := getCredentials(`{"user": "pass"}`)
	s := CreateNewServer(testConfig, creds)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.HTTPMethod, tc.Endpoint, strings.NewReader(tc.Payload))
			response := executeMockRequest(req, s)
			checkResponseCode(t, tc.HTTPStatus, response.Code)
			if tc.ResponseJSON != "" {
				require.JSONEq(t, tc.ResponseJSON, response.Body.String())
			}
			if tc.ResponseText != "" {
				require.Equal(t, tc.ResponseText, response.Body.String())
			}
		})
	}
}
