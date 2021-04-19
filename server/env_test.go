package server

import (
	"net/http"
	"testing"
)

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
