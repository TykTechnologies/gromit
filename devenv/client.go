package devenv

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
)

type GromitClient struct {
	Server    string
	AuthToken string
	Client    http.Client
}

// Replace uses PUT to replace the env
func (g *GromitClient) Replace(name string, body io.Reader) error {
	api := "/env/" + url.PathEscape(name)
	resp, rc, err := g.makeRequest("PUT", api, body, "application/json")
	log.Trace().Bytes("resp", resp).Msg("replace")
	switch rc {
	case http.StatusCreated:
		return nil
	case http.StatusConflict:
		log.Debug().Str("env", name).Msg("exists")
	default:
		err = fmt.Errorf("expected %d response, got %d err: %w", http.StatusCreated, rc, err)
	}
	return err
}

// Delete uses DELETE to delete the env
func (g *GromitClient) Delete(name string) error {
	api := "/env/" + url.PathEscape(name)
	resp, rc, err := g.makeRequest("DELETE", api, nil, "application/json")
	log.Trace().Bytes("resp", resp).Msg("delete")
	if rc != http.StatusAccepted {
		return fmt.Errorf("expected %d response, got %d", http.StatusAccepted, rc)
	}
	return err
}

// Get will return an env map
func (g *GromitClient) Get(name string) (string, error) {
	api := "/env/" + url.PathEscape(name)
	resp, rc, err := g.makeRequest("GET", api, nil, "application/json")
	if err != nil {
		return "", err
	}
	if rc != http.StatusOK {
		return "", fmt.Errorf("%s not found", name)
	}
	var env DevEnv
	err = json.Unmarshal(resp, &env)
	if err != nil {
		return "", fmt.Errorf("unmarshalling %+q: %w", resp, err)
	}
	// Return the JSON, not the map
	return string(resp), nil
}

// makeRequest is a private method that makes the low-level HTTP call to the API
// body can be nil
func (g *GromitClient) makeRequest(method string, api string, body io.Reader, contentType string) ([]byte, int, error) {
	gurl := g.Server + api
	req, err := http.NewRequest(method, gurl, body)
	if err != nil {
		return []byte{}, 0, fmt.Errorf("constructing request to %s: %w", gurl, err)
	}
	if len(g.AuthToken) > 0 {
		req.Header.Add("Authorization", g.AuthToken)
	}
	//req.Header.Add("Content-Type", contentType)
	resp, err := g.Client.Do(req)
	if err != nil {
		return []byte{}, resp.StatusCode, fmt.Errorf("making request to %s: %w", gurl, err)
	}
	defer resp.Body.Close()

	respContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, resp.StatusCode, fmt.Errorf("reading response from %s: %w", gurl, err)
	}
	return respContent, resp.StatusCode, nil
}
