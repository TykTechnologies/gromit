package env

import (
	"bytes"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

const pk = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA13oqkgO3RaYCMUxskU72
S5iBxTsc/KDNgcpoV3nujJuxRHC5jj3+bGaNMfpzMFCdzmtIjdkBnefLiCnqeGlT
CZCK627P1JT9ZRR9R6DGBk5Swr2ZXs0TefIR3HDJmtzBBGj63t9j6VTBYS7fnn2V
3MQG66cszXr6qPUpaN6EK61oGGs4517Ql1BzxGPdC8GJpr9teqgSLuFeeJwyqBqe
CxXxNjZ6OMjWqU2IT+lgUS97UbF1ep8iZJUdvwOmFBoWs6cY9SoTdzlzB4q90Kqs
tapRIa8HM7WWnwmI+i9uGl1QOmZfshOovOgzIZSJh1K43cdFSxgBvpO5ENyLeKai
ZwIDAQAB
-----END PUBLIC KEY-----
`

func TestParseKey(t *testing.T) {
	cases := []struct {
		botResponse string
		expiry      string
		owner       string
		name        string
	}{
		{
			botResponse: "../testdata/dash.trial",
			expiry:      "2020-11-26T10:32:59.860564083Z",
			owner:       "5779711945f92e6689000127",
			name:        "dash",
		},
		{
			botResponse: "../testdata/mdcb.trial",
			expiry:      "2020-11-26T12:30:03.860564083Z",
			owner:       "5779711945f92e6689000127",
			name:        "mdcb",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			botResp, err := os.ReadFile(tc.botResponse)
			if err != nil {
				t.Fatalf("Could not open file: %s\n", tc.botResponse)
			}
			lkey, err := parseKey(bytes.NewReader(botResp))
			if err != nil {
				t.Fatalf("Could not get key: %s\n", botResp)
			}
			token, err := jwt.Parse(string(lkey), func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("Expected token algorithm '%v' but got '%v'", jwt.SigningMethodRS256.Alg(), token.Method)
				}
				return jwt.ParseRSAPublicKeyFromPEM([]byte(pk))
			})
			if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
				t.Fatalf("Malformed license key: %s %v\n", lkey, err)
			}
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if claims["owner"] != tc.owner {
					t.Error("Expected", tc.owner, "got", claims["owner"])
				}
				licenseExpiry := time.Unix(int64(claims["exp"].(float64)), int64(860564083)).UTC()
				expectedExpiry, err := time.Parse(time.RFC3339, tc.expiry)
				if err != nil {
					t.Fatalf("Bad check in test: %s", tc.expiry)
				}
				if licenseExpiry != expectedExpiry {
					t.Error("Expected", expectedExpiry, "got", licenseExpiry)
				}
			} else {
				t.Fatal(ok, token.Valid, claims)
			}
		})
	}
}

func TestFetch(t *testing.T) {
	cases := []struct {
		botResponse string
		name        string
	}{
		{
			botResponse: "../testdata/dash.trial",
			name:        "dash-trial",
		},
		{
			botResponse: "../testdata/mdcb.trial",
			name:        "mdcb-trial",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(T *testing.T) {
			response, err := os.ReadFile(tc.botResponse)
			if err != nil {
				t.Fatalf("Could not find mock response fixture %s", tc.botResponse)
			}
			mockClient, teardown := mockHTTPClient(response)
			defer teardown()
			temp := Licenser{
				Client: mockClient,
			}
			_, err = temp.Fetch("http://this.is.fake/", tc.name, "token")
			if err != nil {
				t.Fatal("Failed fetching", tc.name, err)
			}
		})
	}
}

func mockHTTPClient(response []byte) (*http.Client, func()) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(response)
	})
	s := httptest.NewServer(h)
	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	return cli, s.Close
}
