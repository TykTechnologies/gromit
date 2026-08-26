package pkgs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
	pc "github.com/tyklabs/packagecloud/api/v1"
)

// MirrorStore is the archive the mirror writes to. The sha256 travels
// with each object so a later run (or the deletion step) can confirm
// the archived copy is the artifact the plan announced.
type MirrorStore interface {
	Head(ctx context.Context, key string) (sha256sum string, exists bool, err error)
	Put(ctx context.Context, key, sha256sum string, body io.Reader, length int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// S3Store implements MirrorStore on an S3 bucket
type S3Store struct {
	client *s3.Client
	bucket string
}

func NewS3Store(ctx context.Context, bucket string) (*S3Store, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return &S3Store{client: s3.NewFromConfig(cfg), bucket: bucket}, nil
}

func (s *S3Store) Head(ctx context.Context, key string) (string, bool, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return "", false, nil
		}
		return "", false, err
	}
	return out.Metadata["sha256"], true, nil
}

func (s *S3Store) Put(ctx context.Context, key, sha256sum string, body io.Reader, length int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(length),
		Metadata:      map[string]string{"sha256": sha256sum},
	})
	return err
}

func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

// MirrorResult reports one repo's mirror run. Missing and Failed
// packages mean the plan cannot be executed until resolved: a package
// that is not confirmed archived must never be deleted.
type MirrorResult struct {
	Repo     string   `json:"repo"`
	Mirrored int      `json:"mirrored"`
	Skipped  int      `json:"skipped"`
	Verified int      `json:"verified"`
	Missing  []string `json:"missing,omitempty"`
	Failed   []string `json:"failed,omitempty"`
}

// Clean returns true if every package in the plan is confirmed archived
func (r MirrorResult) Clean() bool {
	return len(r.Missing) == 0 && len(r.Failed) == 0
}

func (r MirrorResult) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d mirrored, %d already archived, %d verified\n",
		r.Repo, r.Mirrored, r.Skipped, r.Verified)
	if len(r.Missing) > 0 {
		fmt.Fprintf(&b, "  MISSING from packagecloud (%d): %s\n", len(r.Missing), strings.Join(r.Missing, ", "))
	}
	if len(r.Failed) > 0 {
		fmt.Fprintf(&b, "  FAILED (%d): %s\n", len(r.Failed), strings.Join(r.Failed, ", "))
	}
	return b.String()
}

// MirrorPlan copies every prune-eligible package in the plan to the
// store. Plan entries are matched against the live repo listing by
// sha256, so a tampered or stale plan cannot cause the wrong bytes to
// be archived. Packages already in the store with a matching checksum
// are skipped, making reruns cheap and idempotent. Nothing is deleted.
func MirrorPlan(ctx context.Context, plan Plan, items []pc.PackageDetail, store MirrorStore, verify bool) MirrorResult {
	res := MirrorResult{Repo: plan.Repo}

	bySha := make(map[string]pc.PackageDetail, len(items))
	for _, item := range items {
		bySha[item.Sha256Sum] = item
	}

	for _, pp := range plan.Packages {
		label := fmt.Sprintf("%s %s %s/%s", pp.Name, pp.Version, pp.DistroVersion, pp.Arch)
		item, found := bySha[pp.Sha256Sum]
		if !found {
			log.Error().Str("sha256", pp.Sha256Sum).Msgf("%s is in the plan but not in the repo", label)
			res.Missing = append(res.Missing, label)
			continue
		}
		key := fmt.Sprintf("%s/%s/%s", plan.Repo, item.DistroVersion, item.Filename)

		sha, exists, err := store.Head(ctx, key)
		if err != nil {
			log.Error().Err(err).Msgf("checking %s", key)
			res.Failed = append(res.Failed, label)
			continue
		}
		switch {
		case exists && sha == pp.Sha256Sum:
			res.Skipped++
		case exists:
			// same key, different content: never overwrite silently
			log.Error().Msgf("%s exists in the archive with sha %s, plan says %s", key, sha, pp.Sha256Sum)
			res.Failed = append(res.Failed, label)
			continue
		default:
			if err := archive(ctx, item, pp.Sha256Sum, key, store); err != nil {
				log.Error().Err(err).Msgf("archiving %s", label)
				res.Failed = append(res.Failed, label)
				continue
			}
			res.Mirrored++
		}
		if verify {
			if err := readBack(ctx, key, pp.Sha256Sum, store); err != nil {
				log.Error().Err(err).Msgf("verifying %s", key)
				res.Failed = append(res.Failed, label)
				continue
			}
			res.Verified++
		}
	}
	sort.Strings(res.Missing)
	sort.Strings(res.Failed)
	return res
}

// archive downloads one package to a temp file, confirms its sha256
// matches the plan, and uploads it
func archive(ctx context.Context, item pc.PackageDetail, wantSha, key string, store MirrorStore) error {
	req, err := http.NewRequestWithContext(ctx, "GET", item.DownloadURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", item.DownloadURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: %s", item.DownloadURL, resp.Status)
	}

	tmp, err := os.CreateTemp("", "mirror-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	hash := sha256.New()
	length, err := io.Copy(io.MultiWriter(tmp, hash), resp.Body)
	if err != nil {
		return fmt.Errorf("saving %s: %w", item.Filename, err)
	}
	gotSha := hex.EncodeToString(hash.Sum(nil))
	if gotSha != wantSha {
		return fmt.Errorf("downloaded sha %s does not match plan sha %s", gotSha, wantSha)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return store.Put(ctx, key, wantSha, tmp, length)
}

// readBack fetches the archived object and confirms its content hash,
// proving the archive copy is restorable
func readBack(ctx context.Context, key, wantSha string, store MirrorStore) error {
	body, err := store.Get(ctx, key)
	if err != nil {
		return err
	}
	defer body.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, body); err != nil {
		return err
	}
	gotSha := hex.EncodeToString(hash.Sum(nil))
	if gotSha != wantSha {
		return fmt.Errorf("archived sha %s does not match plan sha %s", gotSha, wantSha)
	}
	return nil
}
