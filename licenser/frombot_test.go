package licenser

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
)

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
			botResp, err := ioutil.ReadFile(tc.botResponse)
			if err != nil {
				t.Fatalf("Could not open file: %s\n", tc.botResponse)
			}
			lkey, err := parseKey(bytes.NewReader(botResp))
			if err != nil {
				t.Fatalf("Could not get key: %s\n", botResp)
			}
			token, err := jwt.Parse(string(lkey), nil)
			if err.(*jwt.ValidationError).Errors&jwt.ValidationErrorMalformed != 0 {
				t.Fatalf("Malformed license key: %s %v\n", lkey, err)
			}
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if claims["owner"] != tc.owner {
					t.Errorf("Expected owner %s, got %s", tc.owner, claims["owner"])
				}
				licenseExpiry := time.Unix(int64(claims["exp"].(float64)), int64(860564083)).UTC()
				expectedExpiry, err := time.Parse(time.RFC3339, tc.expiry)
				if err != nil {
					t.Fatalf("Bad check in test: %s", tc.expiry)
				}
				if licenseExpiry != expectedExpiry {
					t.Errorf("Expected expiry %s, got %s", expectedExpiry, licenseExpiry)
				}
			} else {
				t.Fatal(ok, token.Valid, claims)
			}
		})
	}
}
