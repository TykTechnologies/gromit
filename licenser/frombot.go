package licenser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// wireStruct models what is returned on the wire
type wireStruct struct {
	attachments []struct {
		text string
	}
	Key string `json:"text"`
}

// Licenser models the public type for this module
type Licenser struct {
	Client *http.Client
}

func (l *Licenser) Fetch(baseURL string, product string, token string) (string, error) {
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return "", err
	}
	req.URL.Path += product
	params := req.URL.Query()
	params.Add("token", token)
	req.URL.RawQuery = params.Encode()

	resp, err := l.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return parseKey(resp.Body)
}

func parseKey(r io.Reader) (string, error) {
	var t wireStruct
	err := json.NewDecoder(r).Decode(&t)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile("```(.+)```")
	matches := re.FindStringSubmatch(string(t.Key))
	if len(matches) < 2 {
		return "", fmt.Errorf("did not find license key in: %s", t.Key)
	}
	return matches[1], nil
}
