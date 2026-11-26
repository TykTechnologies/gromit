package pkgs

import (
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"
)

// RecentImage is one image digest pushed to a Hub org since a cutoff,
// with every tag that points at it, so each image is scanned once.
type RecentImage struct {
	Image      string    `json:"image"`
	Digest     string    `json:"digest"`
	Ref        string    `json:"ref"`
	Tags       []string  `json:"tags"`
	LastPushed time.Time `json:"last_pushed"`
	PushedBy   []string  `json:"pushed_by"`
}

// RecentResult is what RecentImages found. Failed lists the repositories
// that could not be listed, so a partial sweep is reported as partial.
type RecentResult struct {
	Images []RecentImage `json:"images"`
	Failed []string      `json:"failed"`
}

type hubRepoPage struct {
	Next    string        `json:"next"`
	Results []hubRepoInfo `json:"results"`
}

type hubRepoInfo struct {
	Name        string `json:"name"`
	LastUpdated string `json:"last_updated"`
	IsPrivate   bool   `json:"is_private"`
}

// recentTag is a tag entry with the account that pushed it.
type recentTag struct {
	ImageTag
	pushedBy string
}

// RecentImages lists the images pushed to the org's public repositories
// since the cutoff, across all of them, not only those in the gromit
// config: an image pushed by hand to a repo no pipeline knows about is
// the case this exists to catch. Private repositories are left out: a
// secret in them is not exposed to the public, and their names should
// not appear in a public sweep's logs.
//
// Both listings use Hub's ordering=last_updated, which returns the
// newest first, so paging stops at the first entry last updated before
// the cutoff. An entry with no readable time counts as recent. A repo
// whose tags cannot be listed is recorded in Failed and the rest go on.
func (c *HubClient) RecentImages(org string, since time.Time, concurrency int) (RecentResult, error) {
	res := RecentResult{Images: []RecentImage{}, Failed: []string{}}
	if org == "" || strings.Contains(org, "/") {
		return res, fmt.Errorf("org %q must be a single Hub namespace", org)
	}
	if concurrency < 1 {
		return res, fmt.Errorf("concurrency must be at least 1")
	}
	if _, err := c.hubBearer(); err != nil {
		return res, err
	}
	repos, err := c.recentRepos(org, since)
	if err != nil {
		return res, err
	}
	for _, repo := range repos {
		image := org + "/" + repo
		tags, err := c.recentTags(image, since)
		if err != nil {
			res.Failed = append(res.Failed, fmt.Sprintf("%s: %v", image, err))
			continue
		}
		plain := make([]ImageTag, len(tags))
		for i := range tags {
			plain[i] = tags[i].ImageTag
		}
		c.FillMissingDigests(image, plain, concurrency)
		for i := range tags {
			tags[i].Digest = plain[i].Digest
		}
		res.Images = append(res.Images, groupByDigest(image, tags)...)
	}
	return res, nil
}

func (c *HubClient) recentRepos(org string, since time.Time) ([]string, error) {
	u := fmt.Sprintf("%s/v2/namespaces/%s/repositories?page_size=100&ordering=last_updated", c.base, url.PathEscape(org))
	var names []string
	for u != "" {
		var page hubRepoPage
		if err := c.get(u, &page); err != nil {
			return nil, fmt.Errorf("listing %s repositories: %w", org, err)
		}
		for _, r := range page.Results {
			if before(hubTime(r.LastUpdated), since) {
				return names, nil
			}
			if !r.IsPrivate {
				names = append(names, r.Name)
			}
		}
		u = page.Next
	}
	return names, nil
}

func (c *HubClient) recentTags(image string, since time.Time) ([]recentTag, error) {
	path, err := hubImagePath(image)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/v2/repositories/%s/tags?page_size=100&ordering=last_updated", c.base, path)
	var tags []recentTag
	for u != "" {
		var page hubTagPage
		if err := c.get(u, &page); err != nil {
			return nil, err
		}
		for _, e := range page.Results {
			// Stop on the field Hub sorts by; filter on when the tag was pushed.
			if before(hubTime(e.LastUpdated), since) {
				return tags, nil
			}
			pushed := parseHubTime(e)
			if before(pushed, since) {
				continue
			}
			tags = append(tags, recentTag{
				ImageTag: ImageTag{Name: e.Name, Digest: digestFromEntry(e), LastPushed: pushed},
				pushedBy: e.LastUpdater,
			})
		}
		u = page.Next
	}
	return tags, nil
}

func hubTime(s string) time.Time {
	return parseHubTime(hubTagEntry{LastUpdated: s})
}

// before treats an unknown (zero) time as recent, so an entry is listed
// rather than silently skipped.
func before(t, since time.Time) bool {
	return !t.IsZero() && t.Before(since)
}

// groupByDigest merges tags that point at the same digest. A tag whose
// digest could not be found is kept on its own and scanned by tag.
func groupByDigest(image string, tags []recentTag) []RecentImage {
	byKey := map[string]*RecentImage{}
	var keys []string
	for _, t := range tags {
		key, ref := t.Digest, image+"@"+t.Digest
		if t.Digest == "" {
			key, ref = "tag:"+t.Name, image+":"+t.Name
		}
		ri, ok := byKey[key]
		if !ok {
			ri = &RecentImage{Image: image, Digest: t.Digest, Ref: "docker.io/" + ref}
			byKey[key] = ri
			keys = append(keys, key)
		}
		ri.Tags = append(ri.Tags, t.Name)
		if t.LastPushed.After(ri.LastPushed) {
			ri.LastPushed = t.LastPushed
		}
		if t.pushedBy != "" && !slices.Contains(ri.PushedBy, t.pushedBy) {
			ri.PushedBy = append(ri.PushedBy, t.pushedBy)
		}
	}
	out := make([]RecentImage, 0, len(keys))
	for _, k := range keys {
		ri := byKey[k]
		sort.Strings(ri.Tags)
		sort.Strings(ri.PushedBy)
		out = append(out, *ri)
	}
	return out
}
