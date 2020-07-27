package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var a App

func TestMain(m *testing.M) {
	os.Setenv("GROMIT_TABLENAME", "DeveloperEnvironments")
	os.Setenv("GROMIT_REPOS", "tyk,tyk-analytics,tyk-pump")
	os.Setenv("GROMIT_REGISTRYID", "046805072452")
	a.Init("../ccerts/ca.pem")
	code := m.Run()
	os.Exit(code)
}

func TestEmptyTable(t *testing.T) {
	req, _ := http.NewRequest("GET", "/envs", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}

func TestInfraURLs(t *testing.T) {
	// Use this formulation of sub-tests when ordering matters
	// GetLoglvl below works because InfoLvl has set it
	cases := []struct {
		Name             string
		Endpoint         string
		HTTPStatus       int
		ResponseBodyText string
		HTTPMethod       string
	}{
		{
			Name:             "Healthcheck",
			Endpoint:         "/healthcheck",
			HTTPStatus:       http.StatusOK,
			ResponseBodyText: "OK",
			HTTPMethod:       "GET",
		},
		{
			Name:             "InfoLogLvl",
			Endpoint:         "/loglevel/info",
			HTTPStatus:       http.StatusOK,
			ResponseBodyText: `{"level":"info"}`,
			HTTPMethod:       "PUT",
		},
		{
			Name:             "GetLoglvl",
			Endpoint:         "/loglevel",
			HTTPStatus:       http.StatusOK,
			ResponseBodyText: `{"level":"info"}`,
			HTTPMethod:       "GET",
		},
		{
			Name:             "DebugLogLvl",
			Endpoint:         "/loglevel/debug",
			HTTPStatus:       http.StatusOK,
			ResponseBodyText: `{"level":"debug"}`,
			HTTPMethod:       "PUT",
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.HTTPMethod, tc.Endpoint, nil)
			response := executeRequest(req)

			checkResponseCode(t, tc.HTTPStatus, response.Code)

			if body := response.Body.String(); body != tc.ResponseBodyText {
				t.Errorf("Expected %s. Got %s", tc.ResponseBodyText, body)
			}
		})
	}

}

func TestEnv(t *testing.T) {
	cases := []struct {
		Name             string
		Endpoint         string
		Payload          string
		HTTPStatus       int
		ResponseBodyText string
		HTTPMethod       string
		RequestParams    string
	}{
		{
			Name:             "UnknownEnv",
			Endpoint:         "/env/unknown-env",
			HTTPStatus:       http.StatusNotFound,
			ResponseBodyText: `{"error":"does not exist:unknown-env"}`,
			HTTPMethod:       "GET",
		},
		{
			Name:       "InsertTestEnv",
			Endpoint:   "/env/test",
			Payload:    `{"name":"test", "tyk":"sha1", "tyk-analytics":"sha2", "tyk-pump":"sha3"}`,
			HTTPStatus: http.StatusCreated,
			HTTPMethod: "PUT",
		},
		{
			Name:             "GetTestEnv",
			Endpoint:         "/env/test",
			HTTPStatus:       http.StatusOK,
			ResponseBodyText: `{"level":"info"}`,
			HTTPMethod:       "GET",
		},
		{
			Name:             "UpdateTestEnv",
			Endpoint:         "/env/test",
			Payload:          `{"name":"test", "tyk":"updated-sha"}`,
			HTTPStatus:       http.StatusOK,
			ResponseBodyText: `{"level":"debug"}`,
			HTTPMethod:       "PATCH",
		},
		{
			Name:             "CheckUpdate",
			Endpoint:         "/env/test",
			Payload:          `{"name":"test", "tyk":"updated-sha"}`,
			HTTPStatus:       http.StatusOK,
			ResponseBodyText: `{"name":"test", "tyk":"updated-sha", "tyk-analytics":"sha2", "tyk-pump":"sha3"}`,
			HTTPMethod:       "GET",
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.HTTPMethod, tc.Endpoint, strings.NewReader(tc.Payload))
			response := executeRequest(req)

			checkResponseCode(t, tc.HTTPStatus, response.Code)

			if body := response.Body.String(); body != tc.ResponseBodyText {
				t.Errorf("Expected %s. Got %s", tc.ResponseBodyText, body)
			}
		})
	}

	req, _ := http.NewRequest("GET", "/env/doesnotexist", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response.Code)
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
