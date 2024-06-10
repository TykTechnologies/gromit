package policy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

type DockerConfig struct {
	Auths map[string]AuthConfig `json:"auths"`
}

type AuthConfig struct {
	Auth string `json:"auth"`
}

type ParsedImageName struct {
	Registry string
	Repo     string
	Tag      string
}

type Matches struct {
	Repos    map[string]string
	Registry string
}

func (m Matches) Match(repo string) string {
	match, found := m.Repos[repo]
	varName := fmt.Sprintf("%s_image", strings.ReplaceAll(repo, "-", "_"))
	if found {
		return fmt.Sprintf("%s=%s", varName, match)
	} else {
		return fmt.Sprintf("%s=%s/%s:master", varName, m.Registry, repo)
	}
}

func (m Matches) Len() int {
	return len(m.Repos)
}

func ParseImageName(imageName string) *ParsedImageName {
	var registry, repo, tag string

	parts := strings.Split(imageName, "/")
	if len(parts) == 1 {
		// Default registry when no registry part is provided
		registry = "docker.io"
	} else if len(parts) > 1 {
		registry = parts[0]
		parts = parts[1:]
	}
	// Join the remaining parts as the repository name
	repo = strings.Join(parts, "/")

	// Split repo and tag
	if idx := strings.LastIndex(repo, ":"); idx != -1 {
		tag = repo[idx+1:]
		repo = repo[:idx]
	} else {
		tag = "latest"
	}

	return &ParsedImageName{
		Registry: registry,
		Repo:     repo,
		Tag:      tag,
	}
}

func checkTagExists(registry, repo, tag, authHeader string) (bool, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repo, tag)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	} else {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}
}

func NewDockerAuths(fname string) (*DockerConfig, error) {
	data, err := os.ReadFile(os.ExpandEnv(fname))
	if err != nil {
		return nil, err
	}

	var config DockerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (d *DockerConfig) GetMatches(registry, tag string, repos []string) (Matches, error) {
	matches := make(map[string]string)
	auth, err := d.getAuthHeader(registry)
	if err != nil {
		return Matches{}, err
	}
	for _, repo := range repos {
		exists, err := checkTagExists(registry, repo, tag, auth)
		if err != nil {
			log.Warn().Err(err).Msg("checking tag")
			continue
		}
		if exists {
			matches[repo] = fmt.Sprintf("%s/%s:%s", registry, repo, tag)
		}
	}
	return Matches{
		Repos:    matches,
		Registry: registry,
	}, nil
}

func (d *DockerConfig) getAuthHeader(registry string) (string, error) {
	authConfig, exists := d.Auths[registry]
	if !exists {
		return "", fmt.Errorf("no auth config found for registry: %s", registry)
	}

	authDecoded, err := base64.StdEncoding.DecodeString(authConfig.Auth)
	if err != nil {
		return "", err
	}

	authHeader := "Basic " + base64.StdEncoding.EncodeToString(authDecoded)
	return authHeader, nil
}
