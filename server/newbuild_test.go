package server

import (
	"net/http"
	"testing"
)

func TestNewBuild(t *testing.T) {
	cases := []APITestCase{
		{
			Name:       "PlainPump",
			Endpoint:   "/newbuild",
			HTTPStatus: http.StatusOK,
			Payload:    `{"repo":"tyk-pump","ref":"app-test-pump","sha":"sha-pump"}`,
			HTTPMethod: "POST",
		},
		{
			Name:         "CheckPlainPump",
			Endpoint:     "/env/app-test-pump",
			HTTPStatus:   http.StatusOK,
			ResponseJSON: `{"name":"app-test-pump", "portal":"master","state":"new","tyk":"master","tyk-analytics":"master","tyk-identity-broker":"master","tyk-pump":"sha-pump","tyk-sink":"master"}`,
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
