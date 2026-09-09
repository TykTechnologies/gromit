package pkgs

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// ImageArchiveKey is the S3 object for one Hub tag. Restore parses it
// back into image + tag and looks the digest up in images.json.
func ImageArchiveKey(image, tag string) string {
	return fmt.Sprintf("images/%s/%s.tar", image, tag)
}

// ParseImageArchiveKey splits images/org/name/tag.tar. ok is false for
// package keys and anything else.
func ParseImageArchiveKey(key string) (image, tag string, ok bool) {
	rest, found := strings.CutPrefix(key, "images/")
	if !found || rest == "" || !strings.HasSuffix(rest, ".tar") {
		return "", "", false
	}
	rest = strings.TrimSuffix(rest, ".tar")
	slash := strings.LastIndex(rest, "/")
	if slash < 1 || slash == len(rest)-1 {
		return "", "", false
	}
	image, tag = rest[:slash], rest[slash+1:]
	if !strings.Contains(image, "/") {
		return "", "", false
	}
	return image, tag, true
}

// ImageSource pulls one Hub image as an OCI-layout tarball. Tests fake
// it; HubClient talks to the registry.
type ImageSource interface {
	Exists(ctx context.Context, image, digest string) (bool, error)
	Fetch(ctx context.Context, image, tag, digest string) (path string, err error)
	Check(path, digest string) error
}

// MirrorImagePlan copies every prune-eligible Hub tag in the plan to
// the store as an OCI-layout tarball, keyed by image+tag. Entries are
// matched against the live registry by digest, so a stale plan cannot
// archive the wrong image. FIPS plans (NeverMirror) are skipped
// entirely: they may be deleted later, never archived. Reruns are
// idempotent. Nothing is deleted.
func MirrorImagePlan(ctx context.Context, plan ImagePlan, src ImageSource, store MirrorStore, verify bool) MirrorResult {
	res := MirrorResult{Repo: plan.Image, Kind: "images"}
	if plan.NeverMirror {
		res.Skipped = len(plan.Tags)
		return res
	}

	for _, t := range plan.Tags {
		label := plan.Image + ":" + t.Tag
		if t.Digest == "" {
			log.Error().Msgf("%s has an empty digest", label)
			res.Failed = append(res.Failed, label)
			continue
		}
		exists, err := src.Exists(ctx, plan.Image, t.Digest)
		if err != nil {
			log.Error().Err(err).Msgf("checking %s", label)
			res.Failed = append(res.Failed, label)
			continue
		}
		if !exists {
			log.Error().Str("digest", t.Digest).Msgf("%s is in the plan but not on Hub", label)
			res.Missing = append(res.Missing, label)
			continue
		}
		key := ImageArchiveKey(plan.Image, t.Tag)

		sha, found, err := store.Head(ctx, key)
		if err != nil {
			log.Error().Err(err).Msgf("checking %s", key)
			res.Failed = append(res.Failed, label)
			continue
		}
		switch {
		case found && sha == t.Digest:
			res.Skipped++
		case found:
			log.Error().Msgf("%s exists in the archive with digest %s, plan says %s", key, sha, t.Digest)
			res.Failed = append(res.Failed, label)
			continue
		default:
			if err := archiveImage(ctx, src, plan.Image, t.Tag, t.Digest, key, store); err != nil {
				log.Error().Err(err).Msgf("archiving %s", label)
				res.Failed = append(res.Failed, label)
				continue
			}
			res.Mirrored++
		}
		if verify {
			if err := readBackImage(ctx, key, t.Digest, src, store); err != nil {
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

func archiveImage(ctx context.Context, src ImageSource, image, tag, digest, key string, store MirrorStore) error {
	path, err := src.Fetch(ctx, image, tag, digest)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	if err := src.Check(path, digest); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	return store.Put(ctx, key, digest, f, st.Size())
}

func readBackImage(ctx context.Context, key, digest string, src ImageSource, store MirrorStore) error {
	body, err := store.Get(ctx, key)
	if err != nil {
		return err
	}
	defer body.Close()
	tmp, err := os.CreateTemp("", "image-verify-*.tar")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := io.Copy(tmp, body); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return src.Check(tmp.Name(), digest)
}

func (c *HubClient) remoteOptions(ctx context.Context) []remote.Option {
	opts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(&limitedTransport{lim: c.limiter}),
	}
	if c.username != "" && c.token != "" && !isHubJWT(c.token) {
		opts = append(opts, remote.WithAuth(&authn.Basic{Username: c.username, Password: c.token}))
	} else if c.token != "" {
		opts = append(opts, remote.WithAuth(&authn.Bearer{Token: c.token}))
	}
	return opts
}

func (c *HubClient) digestRef(image, digest string) (name.Digest, error) {
	if image == "" || !strings.HasPrefix(digest, "sha256:") {
		return name.Digest{}, fmt.Errorf("need org/name and sha256 digest, got %s@%s", image, digest)
	}
	if c.registry != "" && c.registry != registryAPI {
		host := strings.TrimPrefix(strings.TrimPrefix(c.registry, "https://"), "http://")
		return name.NewDigest(host+"/"+image+"@"+digest, name.Insecure)
	}
	return name.NewDigest(image + "@" + digest)
}

// Exists reports whether the registry still serves this digest.
func (c *HubClient) Exists(ctx context.Context, image, digest string) (bool, error) {
	ref, err := c.digestRef(image, digest)
	if err != nil {
		return false, err
	}
	_, err = remote.Head(ref, c.remoteOptions(ctx)...)
	if err == nil {
		return true, nil
	}
	var te *transport.Error
	if errors.As(err, &te) && te.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

// Fetch writes an OCI-layout tarball of image@digest to a temp file.
// The caller must remove the file.
func (c *HubClient) Fetch(ctx context.Context, image, tag, digest string) (string, error) {
	ref, err := c.digestRef(image, digest)
	if err != nil {
		return "", err
	}
	desc, err := remote.Get(ref, c.remoteOptions(ctx)...)
	if err != nil {
		return "", fmt.Errorf("pulling %s@%s: %w", image, digest, err)
	}
	if desc.Digest.String() != digest {
		return "", fmt.Errorf("registry digest %s does not match plan %s", desc.Digest, digest)
	}

	dir, err := os.MkdirTemp("", "oci-layout-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	p, err := layout.Write(dir, empty.Index)
	if err != nil {
		return "", err
	}
	ann := layout.WithAnnotations(map[string]string{
		"org.opencontainers.image.ref.name": "docker.io/" + image + ":" + tag,
	})
	if desc.MediaType.IsIndex() {
		idx, err := desc.ImageIndex()
		if err != nil {
			return "", err
		}
		if err := p.AppendIndex(idx, ann); err != nil {
			return "", err
		}
	} else {
		img, err := desc.Image()
		if err != nil {
			return "", err
		}
		if err := p.AppendImage(img, ann); err != nil {
			return "", err
		}
	}

	tmp, err := os.CreateTemp("", "image-*.tar")
	if err != nil {
		return "", err
	}
	if err := tarDirectory(dir, tmp); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// Check confirms path is an OCI-layout tarball that contains digest.
func (c *HubClient) Check(path, digest string) error {
	return CheckImageArchive(path, digest)
}

// CheckImageArchive extracts an OCI-layout tarball and confirms it
// contains the plan digest (the index digest for multi-arch tags).
func CheckImageArchive(path, digest string) error {
	dir, err := os.MkdirTemp("", "oci-check-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := untar(path, dir); err != nil {
		return err
	}
	idx, err := layout.ImageIndexFromPath(dir)
	if err != nil {
		return fmt.Errorf("reading OCI layout: %w", err)
	}
	found, err := indexHasDigest(idx, digest)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("archive does not contain digest %s", digest)
	}
	return nil
}

func indexHasDigest(idx v1.ImageIndex, want string) (bool, error) {
	mf, err := idx.IndexManifest()
	if err != nil {
		return false, err
	}
	for _, desc := range mf.Manifests {
		if desc.Digest.String() == want {
			return true, nil
		}
		if !desc.MediaType.IsIndex() {
			continue
		}
		nested, err := idx.ImageIndex(desc.Digest)
		if err != nil {
			return false, err
		}
		ok, err := indexHasDigest(nested, want)
		if ok || err != nil {
			return ok, err
		}
	}
	return false, nil
}

type limitedTransport struct {
	lim *rate.Limiter
}

func (t *limitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.lim != nil {
		if err := t.lim.Wait(req.Context()); err != nil {
			return nil, err
		}
	}
	return http.DefaultTransport.RoundTrip(req)
}

func tarDirectory(dir string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if d.IsDir() {
			hdr.Name += "/"
			return tw.WriteHeader(hdr)
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

func untar(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	dest = filepath.Clean(dest)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(hdr.Name)
		if name == "." || strings.HasPrefix(name, "..") || strings.Contains(name, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("refusing path %s", hdr.Name)
		}
		target := filepath.Join(dest, name)
		if !strings.HasPrefix(target, dest+string(os.PathSeparator)) {
			return fmt.Errorf("refusing path %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported tar entry %s", hdr.Name)
		}
	}
}
