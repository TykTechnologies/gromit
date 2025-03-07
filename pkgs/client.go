package pkgs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/TykTechnologies/gromit/util"
	"github.com/peterhellberg/link"
	"github.com/rs/zerolog/log"
	bar "github.com/schollz/progressbar/v3"
	"github.com/spf13/viper"
	pc "github.com/tyklabs/packagecloud/api/v1"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

type Client struct {
	token   string
	owner   string
	limiter *rate.Limiter
	ctx     context.Context
}

func NewClient(authToken, owner string, rps float64, burst int) *Client {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	return &Client{authToken, owner, limiter, context.TODO()}
}

type pkgConfig struct {
	Exceptions    []string
	VersionCutoff string
	AgeCutoff     time.Duration
	NotBackup     bool
}

// CleanConfig is the consolidated options that can be passed to the Clean method
type CleanConfig struct {
	Concurrency int
	Savedir     string
	Delete      bool
	Backup      bool
	RepoName    string
}

// Repos models the config file
type Repos map[string]pkgConfig

type pkgList chan pc.PackageDetail

const pcPrefix = "https://packagecloud.io"

// LoadPkgs returns a map of repos→config from the embedded or
// supplied config file
func LoadConfig() (*Repos, error) {
	pkgs := make(Repos)
	return &pkgs, viper.UnmarshalKey("pkgs", &pkgs)
}

// get makes a GET request to a packagecloud API and returns the next
// page link. Requests are limited by the rate limiter set when the
// client is initialised.
func (c *Client) get(url string) (*http.Response, error, string) {
	var buf bytes.Buffer
	req, err := http.NewRequestWithContext(c.ctx, "GET", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("http newrequest err: %v", err), ""
	}
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Accept", "application/json")

	req.SetBasicAuth(c.token, "")
	err = c.limiter.Wait(c.ctx)
	if err != nil {
		return nil, err, ""
	}
	resp, err := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("invalid response: %s err: %q", resp.Status, b), ""
	}
	total := resp.Header.Get("Total")
	perPage := resp.Header.Get("Per-Page")
	totalInt, _ := strconv.Atoi(total)
	perPageInt, _ := strconv.Atoi(perPage)

	if total != "" && perPage != "" && totalInt > perPageInt {
		webLink := link.ParseResponse(resp)
		if n, ok := webLink["next"]; ok {
			return resp, nil, n.URI
		}

	}
	return resp, nil, ""
}

// AllPackages returns an errgroup that can be used by the caller to
// collect errors and a channel to read PackageDetails from. Only
// items that satisy the supplied filter are written to the channel.
func (c *Client) AllPackages(repo string, filter *Filter) (pkgList, *errgroup.Group) {
	ch := make(pkgList)
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/packages.json", pcPrefix, c.owner, repo)
	pkgs := new(errgroup.Group)
	pkgs.Go(func() error {
		defer close(ch)
		// do until next is ""
		for {
			resp, err, next := c.get(url)
			if err != nil {
				return fmt.Errorf("http get err: %v", err)
			}
			var items []pc.PackageDetail
			err = json.NewDecoder(resp.Body).Decode(&items)
			if err != nil {
				return fmt.Errorf("json parse err: %v", err)
			}
			for _, item := range items {
				filter.IncTotal()
				if filter.Satisfies(item, time.Now()) {
					ch <- item
					filter.IncFiltered()
				}

			}
			resp.Body.Close()
			if next == "" {
				break
			}
			url = next
		}
		return nil
	})
	return ch, pkgs
}

// download a package into savedir/name/distro/ if not already downloaded
func (c *Client) download(item pc.PackageDetail, savedir string) error {
	dirpath := fmt.Sprintf("%s/%s/%s", savedir, item.Name, item.DistroVersion)
	err := os.MkdirAll(dirpath, 0755)
	if err != nil {
		return fmt.Errorf("could not create %s: %v", dirpath, err)
	}
	dir, err := os.OpenRoot(dirpath)
	if err != nil {
		return fmt.Errorf("could not open %s: %v", dirpath, err)
	}
	var sha256sum string
	f, err := dir.Open(item.Filename)
	if err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Msgf("open %s/%s", dirpath, item.Filename)
	}
	defer f.Close()
	if err == nil {
		sha256sum = util.Sha256Sum(f)
	}
	if sha256sum != item.Sha256Sum {
		log.Debug().Msgf("downloading %s/%s as it's hash %s does not match repo hash %s", dirpath, item.Filename, sha256sum, item.Sha256Sum)
		f, err := dir.Create(item.Filename)
		if err != nil {
			return fmt.Errorf("could not create %s/%s: %v", dirpath, item.Filename, err)
		}
		resp, err := http.Get(item.DownloadURL)
		if err != nil {
			return fmt.Errorf("failed to download %s: %v", item.DownloadURL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status fetching %s: %s", item.DownloadURL, resp.Status)
		}

		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write %s/%s: %v", dirpath, item.Filename, err)
		}
	} else {
		log.Debug().Msgf("not downloading %s/%s as it is already downloaded", dirpath, item.Filename)
	}

	return nil
}

// delete deletes the given package from the repo permanently
func (c *Client) delete(item pc.PackageDetail) error {
	var buf bytes.Buffer
	purl, err := url.JoinPath(pcPrefix, item.DestroyURL)
	if err != nil {
		return fmt.Errorf("creating URL: %v", err)
	}
	req, err := http.NewRequestWithContext(c.ctx, "DELETE", purl, &buf)
	if err != nil {
		return fmt.Errorf("http newrequest err: %v", err)
	}
	req.SetBasicAuth(c.token, "")
	err = c.limiter.Wait(c.ctx)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http err %s", purl)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid response: %s err: %q for %s", resp.Status, b, purl)
	}
	log.Debug().Msgf("deleted %s", purl)
	return nil
}

// Clean optionally backs up and then removes the packages from packagecloud
func (c *Client) Clean(pList pkgList, cc CleanConfig) error {
	errChan := make(chan error)
	defer close(errChan)
	progress := bar.Default(-1, cc.RepoName)
	defer progress.Finish()
	pkgs := new(errgroup.Group)
	for range cc.Concurrency {
		pkgs.Go(func() error {
			for item := range pList {
				//fmt.Println(item.Name, item.DistroVersion, item.Filename, item.Sha256Sum)
				progress.Add(1)
				if cc.Backup {
					err := c.download(item, cc.Savedir)
					if err != nil {
						errChan <- fmt.Errorf("download err: %v", err)
						continue
					}
				}
				if cc.Delete {
					err := c.delete(item)
					if err != nil {
						errChan <- fmt.Errorf("delete err: %v", err)
					}
				}
			}
			return nil
		})
	}
	go func() {
		for cleanErr := range errChan {
			log.Error().Err(cleanErr).Msg("while cleaning")
		}
	}()

	if err := pkgs.Wait(); err != nil {
		return err
	}
	return nil
}
