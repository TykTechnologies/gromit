package pkgs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

const (
	hubAPI         = "https://hub.docker.com"
	registryAPI    = "https://registry-1.docker.io"
	registryAuth   = "https://auth.docker.io"
	manifestAccept = "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.v1+json"
)

// ImageTag is one Hub tag. Digest is the identity later stages verify.
type ImageTag struct {
	Name       string
	Digest     string
	LastPushed time.Time
}

// ImagePlan is the Docker Hub equivalent of Plan: same cutoffs, tag+digest
// instead of filename+sha256. Archive keys are images/<image>/<tag>.tar.
type ImagePlan struct {
	Repo        string    `json:"repo"`
	Image       string    `json:"image"`
	GeneratedAt time.Time `json:"generated_at"`
	NotBefore   time.Time `json:"not_before"`
	Track       string    `json:"track,omitempty"`
	Editions    []string  `json:"editions,omitempty"`
	Product     string    `json:"product,omitempty"`
	Anchor      string    `json:"anchor,omitempty"`
	Cutoff      string    `json:"cutoff,omitempty"`
	Series      []string  `json:"series"`

	Retained int `json:"retained"`
	Pruned   int `json:"pruned"`
	// NeverMirror is set for FIPS images: they may be deleted, never archived
	NeverMirror bool `json:"never_mirror,omitempty"`
	// NonRelease counts aliases, RCs, arch suffixes and moving tags;
	// they are retained until an explicit rule covers them.
	NonRelease int `json:"non_release"`

	PrunedSeries map[string]int `json:"pruned_series,omitempty"`
	Protected    map[string]int `json:"protected,omitempty"`

	Tags []PlanImage `json:"tags,omitempty"`
}

// PlanImage identifies one prune-eligible release tag
type PlanImage struct {
	Tag        string    `json:"tag"`
	Digest     string    `json:"digest"`
	LastPushed time.Time `json:"last_pushed"`
}

// IsReleaseTag is a real version name: vMAJOR.MINOR.PATCH, nothing else.
func IsReleaseTag(tag string) bool {
	if !semver.IsValid(tag) {
		return false
	}
	return semver.Canonical(tag) == tag && semver.Prerelease(tag) == ""
}

// HubClient talks to the Docker Hub HTTP API and the registry
type HubClient struct {
	base     string
	registry string
	auth     string
	token    string
	username string
	hubJWT   string
	// hubAnonOnly is set after Hub rejects credentials (typical for
	// org access tokens). Further Hub GETs skip the doomed authed try.
	hubAnonOnly bool
	limiter     *rate.Limiter
	ctx         context.Context
}

func NewHubClient(token string, rps float64, burst int) *HubClient {
	if token == "" {
		token = os.Getenv("DOCKERHUB_TOKEN")
	}
	token = strings.Trim(strings.TrimSpace(token), `"'`)
	username := strings.Trim(strings.TrimSpace(os.Getenv("DOCKERHUB_USERNAME")), `"'`)
	return &HubClient{
		base:     hubAPI,
		registry: registryAPI,
		auth:     registryAuth,
		token:    token,
		username: username,
		limiter:  rate.NewLimiter(rate.Limit(rps), burst),
		ctx:      context.TODO(),
	}
}

type hubTagPage struct {
	Next    string        `json:"next"`
	Results []hubTagEntry `json:"results"`
}

type hubImage struct {
	Digest       string `json:"digest"`
	Architecture string `json:"architecture"`
}

type hubTagEntry struct {
	Name          string     `json:"name"`
	Digest        string     `json:"digest"`
	TagLastPushed string     `json:"tag_last_pushed"`
	LastUpdated   string     `json:"last_updated"`
	Images        []hubImage `json:"images"`
}

func digestFromEntry(e hubTagEntry) string {
	if e.Digest != "" {
		return e.Digest
	}
	var found []string
	for _, img := range e.Images {
		if img.Digest == "" || img.Architecture == "unknown" {
			continue
		}
		found = append(found, img.Digest)
	}
	if len(found) == 1 {
		return found[0]
	}
	return ""
}

func isHubJWT(token string) bool {
	return strings.HasPrefix(token, "eyJ")
}

// hubBearer is the Authorization value for hub.docker.com. A Personal
// Access Token cannot be sent as Bearer; it must be exchanged for a JWT.
func (c *HubClient) hubBearer() (string, error) {
	if c.token == "" {
		return "", nil
	}
	if isHubJWT(c.token) {
		return "Bearer " + c.token, nil
	}
	if c.hubJWT != "" {
		return "Bearer " + c.hubJWT, nil
	}
	if c.username == "" {
		return "", fmt.Errorf("DOCKERHUB_TOKEN is a Hub PAT; set DOCKERHUB_USERNAME to the account that created it (Hub will 401 if the PAT is sent as Bearer). Unset DOCKERHUB_TOKEN to list public repos anonymously")
	}
	jwt, err := c.createAccessToken()
	if err != nil {
		return "", err
	}
	c.hubJWT = jwt
	return "Bearer " + jwt, nil
}

func (c *HubClient) createAccessToken() (string, error) {
	body, err := json.Marshal(map[string]string{
		"identifier": c.username,
		"secret":     c.token,
	})
	if err != nil {
		return "", err
	}
	raw := c.base + "/v2/auth/token"
	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, raw, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.limiter.Wait(c.ctx); err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("hub login: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hub login %s: %s (check DOCKERHUB_USERNAME matches the PAT)", raw, resp.Status)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		Token       string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("hub login: %w", err)
	}
	jwt := out.AccessToken
	if jwt == "" {
		jwt = out.Token
	}
	if jwt == "" {
		return "", fmt.Errorf("hub login: empty access token")
	}
	return jwt, nil
}

func (c *HubClient) authorize(req *http.Request) error {
	raw := req.URL.String()
	if strings.HasPrefix(raw, c.base) {
		hdr, err := c.hubBearer()
		if err != nil {
			return err
		}
		if hdr != "" {
			req.Header.Set("Authorization", hdr)
		}
		return nil
	}
	if c.username != "" && c.token != "" && !isHubJWT(c.token) {
		req.SetBasicAuth(c.username, c.token)
	} else if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return nil
}

func (c *HubClient) get(rawURL string, dest any) error {
	withHubAuth := strings.HasPrefix(rawURL, c.base) && !c.hubAnonOnly
	if !strings.HasPrefix(rawURL, c.base) {
		withHubAuth = true
	}
	return c.doGet(rawURL, dest, withHubAuth)
}

func (c *HubClient) doGet(rawURL string, dest any, withHubAuth bool) error {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	if withHubAuth || !strings.HasPrefix(rawURL, c.base) {
		if err := c.authorize(req); err != nil {
			return err
		}
	}
	if err := c.limiter.Wait(c.ctx); err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) &&
		withHubAuth && c.token != "" && strings.HasPrefix(rawURL, c.base) {
		c.hubAnonOnly = true
		log.Warn().Msgf("Hub rejected credentials (%s); listing public tags without them", resp.Status)
		return c.doGet(rawURL, dest, false)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", rawURL, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func parseHubTime(entry hubTagEntry) time.Time {
	for _, s := range []string{entry.TagLastPushed, entry.LastUpdated} {
		if s == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func hubImagePath(image string) (string, error) {
	org, name, ok := strings.Cut(image, "/")
	if !ok || org == "" || name == "" || strings.Contains(name, "/") {
		return "", fmt.Errorf("dhrepo %q must be org/name", image)
	}
	return org + "/" + name, nil
}

// ListTags fetches every tag. Prefer the Hub list (it includes last
// pushed and often the digest). If Hub rate-limits anonymous paging,
// fall back to the registry tags API, which accepts an org token.
func (c *HubClient) ListTags(image string) ([]ImageTag, error) {
	if _, err := c.hubBearer(); err != nil {
		return nil, err
	}
	tags, err := c.listHubTags(image)
	if err == nil {
		return tags, nil
	}
	log.Warn().Err(err).Msgf("Hub tag list failed for %s, trying the registry", image)
	return c.listRegistryTags(image)
}

func (c *HubClient) listHubTags(image string) ([]ImageTag, error) {
	path, err := hubImagePath(image)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/v2/repositories/%s/tags?page_size=100", c.base, path)
	var all []ImageTag
	for u != "" {
		var page hubTagPage
		if err := c.get(u, &page); err != nil {
			return nil, fmt.Errorf("listing %s: %w", image, err)
		}
		for _, e := range page.Results {
			all = append(all, ImageTag{
				Name:       e.Name,
				Digest:     digestFromEntry(e),
				LastPushed: parseHubTime(e),
			})
		}
		u = page.Next
	}
	return all, nil
}

type registryTagPage struct {
	Tags []string `json:"tags"`
}

func (c *HubClient) listRegistryTags(image string) ([]ImageTag, error) {
	path, err := hubImagePath(image)
	if err != nil {
		return nil, err
	}
	token, err := c.registryToken(path)
	if err != nil {
		return nil, fmt.Errorf("listing %s via registry: %w", image, err)
	}
	u := fmt.Sprintf("%s/v2/%s/tags/list?n=100", c.registry, path)
	var all []ImageTag
	seen := map[string]bool{}
	for u != "" {
		var page registryTagPage
		next, err := c.registryJSON(u, token, &page)
		if err != nil {
			return nil, fmt.Errorf("listing %s via registry: %w", image, err)
		}
		for _, name := range page.Tags {
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			all = append(all, ImageTag{Name: name})
		}
		u = next
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("listing %s via registry: no tags", image)
	}
	return all, nil
}

func (c *HubClient) registryJSON(raw, token string, dest any) (next string, err error) {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if err := c.limiter.Wait(c.ctx); err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", raw, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return "", err
	}
	return nextLink(resp), nil
}

func nextLink(resp *http.Response) string {
	for _, v := range resp.Header.Values("Link") {
		if !strings.Contains(v, `rel="next"`) && !strings.Contains(v, "rel=next") {
			continue
		}
		start := strings.Index(v, "<")
		end := strings.Index(v, ">")
		if start < 0 || end <= start {
			continue
		}
		ref := v[start+1 : end]
		if resp.Request == nil || resp.Request.URL == nil {
			return ref
		}
		u, err := resp.Request.URL.Parse(ref)
		if err != nil {
			return ref
		}
		return u.String()
	}
	return ""
}

// FillPlanDigests looks up digests only for prune-eligible tags the
// list endpoint left blank. Returns an error if any remain empty.
func (c *HubClient) FillPlanDigests(image string, plan *ImagePlan, concurrency int) error {
	tags := make([]ImageTag, len(plan.Tags))
	for i, t := range plan.Tags {
		tags[i] = ImageTag{Name: t.Tag, Digest: t.Digest}
	}
	c.FillMissingDigests(image, tags, concurrency)
	var missing []string
	for i := range plan.Tags {
		plan.Tags[i].Digest = tags[i].Digest
		if plan.Tags[i].Digest == "" {
			missing = append(missing, plan.Tags[i].Tag)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s: no digest for %s", image, strings.Join(missing, ", "))
	}
	return nil
}

// FillMissingDigests looks up tags the list endpoint left blank.
// Prefer the Hub tag digest, then a single per-arch digest, then the
// registry manifest digest (needed for old multi-arch tags).
func (c *HubClient) FillMissingDigests(image string, tags []ImageTag, concurrency int) {
	path, err := hubImagePath(image)
	if err != nil {
		return
	}
	regToken, err := c.registryToken(path)
	if err != nil {
		log.Warn().Err(err).Msgf("registry token for %s", image)
	}
	g := new(errgroup.Group)
	g.SetLimit(concurrency)
	for i := range tags {
		if tags[i].Digest != "" {
			continue
		}
		g.Go(func() error {
			raw := fmt.Sprintf("%s/v2/repositories/%s/tags/%s", c.base, path, url.PathEscape(tags[i].Name))
			var e hubTagEntry
			if err := c.get(raw, &e); err != nil {
				log.Warn().Err(err).Msgf("digest for %s:%s", image, tags[i].Name)
			}
			d := digestFromEntry(e)
			if d == "" && regToken != "" {
				d, err = c.registryDigest(path, tags[i].Name, regToken)
				if err != nil {
					log.Warn().Err(err).Msgf("registry digest for %s:%s", image, tags[i].Name)
				}
			}
			tags[i].Digest = d
			return nil
		})
	}
	_ = g.Wait()
}

func (c *HubClient) registryToken(path string) (string, error) {
	u := fmt.Sprintf("%s/token?service=registry.docker.io&scope=repository:%s:pull", c.auth, path)
	var body struct {
		Token string `json:"token"`
	}
	if err := c.get(u, &body); err != nil {
		return "", err
	}
	if body.Token == "" {
		return "", fmt.Errorf("empty registry token")
	}
	return body.Token, nil
}

func (c *HubClient) registryDigest(path, tag, token string) (string, error) {
	raw := fmt.Sprintf("%s/v2/%s/manifests/%s", c.registry, path, url.PathEscape(tag))
	req, err := http.NewRequestWithContext(c.ctx, http.MethodHead, raw, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", manifestAccept)
	if err := c.limiter.Wait(c.ctx); err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", raw, resp.Status)
	}
	d := resp.Header.Get("Docker-Content-Digest")
	if d == "" {
		return "", fmt.Errorf("%s: no Docker-Content-Digest", raw)
	}
	return d, nil
}

// BuildImagePlan classifies Hub tags with the same cutoffs as BuildPlan.
// Only release tags (vMAJOR.MINOR.PATCH) can be pruned.
func isFIPSImage(image string) bool {
	return strings.HasSuffix(image, "-fips")
}

// ResolveImageArg maps a pkgs key or a Hub repo name to the product
// config and the images to plan. "tyk-gateway-ee" finds the EE image
// under the tyk-gateway pkgs entry.
func (r Repos) ResolveImageArg(arg string) (string, pkgConfig, []string, error) {
	if cfg, found := r[arg]; found {
		images := cfg.HubImages()
		if len(images) == 0 {
			return "", cfg, nil, fmt.Errorf("%s has no dhrepo in pkgs config", arg)
		}
		return arg, cfg, images, nil
	}
	for name, cfg := range r {
		for _, image := range cfg.HubImages() {
			_, short, _ := strings.Cut(image, "/")
			if image == arg || short == arg {
				return name, cfg, []string{image}, nil
			}
		}
	}
	return "", pkgConfig{}, nil, fmt.Errorf("%s not present in pkgs config", arg)
}

func BuildImagePlan(repoName, image string, cfg pkgConfig, tracks Tracks, tags []ImageTag, now time.Time, grace time.Duration) (ImagePlan, error) {
	p := ImagePlan{
		Repo:         repoName,
		Image:        image,
		NeverMirror:  isFIPSImage(image),
		GeneratedAt:  now,
		NotBefore:    now.Add(grace),
		Track:        cfg.Track,
		Editions:     cfg.Editions,
		Product:      cfg.Name,
		PrunedSeries: make(map[string]int),
		Protected:    make(map[string]int),
	}

	var releaseVers []string
	for _, tag := range tags {
		if IsReleaseTag(tag.Name) {
			releaseVers = append(releaseVers, tag.Name)
		}
	}
	p.Series = MinorSeries(releaseVers)

	cutoff := semver.Canonical(cfg.VersionCutoff)
	bySeries := false
	if cfg.Track != "" {
		track, found := tracks[cfg.Track]
		if !found {
			return p, fmt.Errorf("track %q is not in the tracks config", cfg.Track)
		}
		anchor, depth, err := track.Anchor(cfg.Editions)
		if err != nil {
			return p, fmt.Errorf("track %q: %w", cfg.Track, err)
		}
		p.Anchor = anchor
		cutoff, err = DeriveCutoff(p.Series, anchor, depth)
		if err != nil {
			return p, err
		}
		bySeries = true
	}
	p.Cutoff = cutoff

	exceptions := make(map[string]bool)
	for _, e := range cfg.Exceptions {
		exceptions[e] = true
	}

	for _, tag := range tags {
		if !IsReleaseTag(tag.Name) {
			p.NonRelease++
			p.Retained++
			continue
		}
		if exceptions[tag.Name] {
			p.Protected[tag.Name]++
			p.Retained++
			continue
		}
		prune := false
		if cutoff != "" {
			if bySeries {
				prune = semver.Compare(semver.MajorMinor(tag.Name), cutoff) < 0
			} else {
				prune = semver.Compare(tag.Name, cutoff) < 0
			}
		}
		if !prune && cfg.AgeCutoff != 0 && !tag.LastPushed.IsZero() && now.Sub(tag.LastPushed) > cfg.AgeCutoff {
			prune = true
		}
		if prune {
			p.Pruned++
			p.PrunedSeries[semver.MajorMinor(tag.Name)]++
			p.Tags = append(p.Tags, PlanImage{
				Tag:        tag.Name,
				Digest:     tag.Digest,
				LastPushed: tag.LastPushed,
			})
		} else {
			p.Retained++
		}
	}
	return p, nil
}

func (p ImagePlan) Render() string {
	var b strings.Builder
	label := p.Repo
	if p.Image != "" {
		label = fmt.Sprintf("%s (%s)", p.Repo, p.Image)
	}
	fmt.Fprintf(&b, "%s: %d tags, %d retained, %d pruned\n",
		label, p.Retained+p.Pruned, p.Retained, p.Pruned)
	if p.NeverMirror {
		fmt.Fprintf(&b, "  FIPS: will be deleted, never archived\n")
	}
	fmt.Fprintf(&b, "  no deletion before %s\n", p.NotBefore.Format("2006-01-02"))
	if p.Track != "" {
		fmt.Fprintf(&b, "  track %s editions %v, anchor %s -> cutoff %s (oldest retained series)\n",
			p.Track, p.Editions, p.Anchor, p.Cutoff)
	} else if p.Cutoff != "" {
		fmt.Fprintf(&b, "  static cutoff %s\n", p.Cutoff)
	} else if p.Pruned > 0 {
		fmt.Fprintf(&b, "  age-based pruning\n")
	}
	if len(p.PrunedSeries) > 0 {
		series := make([]string, 0, len(p.PrunedSeries))
		for s := range p.PrunedSeries {
			series = append(series, s)
		}
		semver.Sort(series)
		fmt.Fprintf(&b, "  pruned series:")
		for _, s := range series {
			fmt.Fprintf(&b, " %s(%d)", s, p.PrunedSeries[s])
		}
		fmt.Fprintln(&b)
	}
	if len(p.Protected) > 0 {
		keys := make([]string, 0, len(p.Protected))
		for k := range p.Protected {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(&b, "  exceptions held:")
		for _, k := range keys {
			fmt.Fprintf(&b, " %s(%d)", k, p.Protected[k])
		}
		fmt.Fprintln(&b)
	}
	if p.NonRelease > 0 {
		fmt.Fprintf(&b, "  %d non-release tags always retained\n", p.NonRelease)
	}
	return b.String()
}
