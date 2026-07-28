package dhivex

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	advisoriesRepository = "https://github.com/docker-hardened-images/advisories"
	maxDownloadSize      = 128 << 20

	defaultDHIKeyURL    = "https://registry.scout.docker.com/keyring/dhi/latest"
	defaultDHIKeySHA256 = "1d02bbccf149283ae6288d96264dcad3fb23ee1911d90324a48eab28e4cb8a5f"
)

var (
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
	digestPattern = regexp.MustCompile(`@sha256:([0-9a-f]{64})$`)
)

type Options struct {
	Image         string
	Platform      string
	Severity      string
	OutputDir     string
	Timeout       string
	AdvisoriesRef string
	GitHubToken   string
	TrivyPath     string
	CosignPath    string
	HTTPClient    *http.Client
	Log           io.Writer
	Now           func() time.Time
}

type Summary struct {
	Image              string `json:"image"`
	ImageDigest        string `json:"image_digest"`
	PlatformImageID    string `json:"platform_image_id"`
	DHIProduct         string `json:"dhi_product"`
	DHIChainID         string `json:"dhi_chain_id"`
	DHIBaseLayers      int    `json:"dhi_base_layers"`
	Platform           string `json:"platform"`
	Severity           string `json:"severity"`
	AdvisoriesCommit   string `json:"advisories_commit"`
	VerifiedDocuments  int    `json:"verified_documents"`
	ProjectedFindings  int    `json:"projected_findings"`
	UnmatchedFindings  int    `json:"unmatched_findings"`
	ActiveFindings     int    `json:"active_findings"`
	ActiveCritical     int    `json:"active_critical"`
	ActiveHigh         int    `json:"active_high"`
	SuppressedFindings int    `json:"suppressed_findings"`
	SuppressedCritical int    `json:"suppressed_critical"`
	SuppressedHigh     int    `json:"suppressed_high"`
	OutputDir          string `json:"output_dir"`
}

type evidenceManifest struct {
	SchemaVersion   int                  `json:"schema_version"`
	GeneratedAt     string               `json:"generated_at"`
	Image           string               `json:"image"`
	ImageDigest     string               `json:"image_digest"`
	PlatformImageID string               `json:"platform_image_id"`
	Lineage         manifestLineage      `json:"dhi_lineage"`
	Platform        string               `json:"platform"`
	Severity        string               `json:"severity"`
	Timeout         string               `json:"timeout"`
	Advisories      manifestAdvisories   `json:"advisories"`
	Documents       []manifestDocument   `json:"documents"`
	Projections     []manifestProjection `json:"projections"`
	Unmatched       []unmatchedFinding   `json:"unmatched"`
	Reports         manifestReports      `json:"reports"`
}

type manifestAdvisories struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	KeyURL     string `json:"key_url"`
	KeySHA256  string `json:"key_sha256"`
}

type manifestLineage struct {
	Product     string   `json:"product"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Definition  string   `json:"definition"`
	ChainID     string   `json:"chain_id"`
	BaseLayers  int      `json:"base_layers"`
	BaseDiffIDs []string `json:"base_diff_ids"`
}

type manifestDocument struct {
	Product            string `json:"product"`
	URL                string `json:"url"`
	SignatureURL       string `json:"signature_url"`
	SHA256             string `json:"sha256"`
	SignatureSHA256    string `json:"signature_sha256"`
	VerificationSHA256 string `json:"cosign_verification_sha256"`
}

type manifestProjection struct {
	VulnerabilityID string `json:"vulnerability_id"`
	BinaryPackage   string `json:"binary_package"`
	BinaryVersion   string `json:"binary_version"`
	BinaryPURL      string `json:"binary_purl"`
	SourcePackage   string `json:"source_package"`
	SourceVersion   string `json:"source_version"`
	SourcePURL      string `json:"source_purl"`
	SourceDocument  string `json:"source_document"`
	SourceStatement string `json:"source_statement"`
	Status          string `json:"status"`
	Justification   string `json:"justification,omitempty"`
	StatementTime   string `json:"statement_timestamp"`
}

type manifestReports struct {
	TrivyVersionSHA256  string            `json:"trivy_version_sha256"`
	DatabaseFilesSHA256 map[string]string `json:"database_files_sha256"`
	RawSHA256           string            `json:"raw_sha256"`
	SBOMSHA256          string            `json:"sbom_sha256"`
	CompatibilityVEX    string            `json:"compatibility_vex_sha256"`
	FinalSHA256         string            `json:"final_sha256"`
}

type unmatchedFinding struct {
	VulnerabilityID string `json:"vulnerability_id"`
	Package         string `json:"package"`
	PURL            string `json:"purl,omitempty"`
	Reason          string `json:"reason"`
}

type vexDocument struct {
	Context    string         `json:"@context"`
	ID         string         `json:"@id"`
	Author     string         `json:"author"`
	Role       string         `json:"role"`
	Version    int            `json:"version"`
	Timestamp  string         `json:"timestamp"`
	Statements []vexStatement `json:"statements"`
}

type vexStatement struct {
	ID              string           `json:"@id"`
	Vulnerability   vexVulnerability `json:"vulnerability"`
	Products        []vexProduct     `json:"products"`
	Status          string           `json:"status"`
	StatusNotes     string           `json:"status_notes,omitempty"`
	Justification   string           `json:"justification,omitempty"`
	ImpactStatement string           `json:"impact_statement,omitempty"`
	ActionStatement string           `json:"action_statement,omitempty"`
	Timestamp       string           `json:"timestamp,omitempty"`
}

type vexVulnerability struct {
	Name string `json:"name"`
}

type vexProduct struct {
	ID string `json:"@id"`
}

type verifiedVEX struct {
	Product  string
	URL      string
	Document vexDocument
}

type trivyReport struct {
	ArtifactName string        `json:"ArtifactName"`
	ArtifactType string        `json:"ArtifactType"`
	Metadata     trivyMetadata `json:"Metadata"`
	Results      []trivyResult `json:"Results"`
}

type trivyMetadata struct {
	ImageID     string           `json:"ImageID"`
	RepoDigests []string         `json:"RepoDigests"`
	Reference   string           `json:"Reference"`
	DiffIDs     []string         `json:"DiffIDs"`
	OS          trivyOS          `json:"OS"`
	ImageConfig trivyImageConfig `json:"ImageConfig"`
}

type trivyOS struct {
	Family string `json:"Family"`
	Name   string `json:"Name"`
}

type trivyImageConfig struct {
	Architecture string               `json:"architecture"`
	OS           string               `json:"os"`
	Config       trivyContainerConfig `json:"config"`
}

type trivyContainerConfig struct {
	Labels map[string]string `json:"Labels"`
}

type trivyResult struct {
	Type                         string                 `json:"Type"`
	Vulnerabilities              []trivyVulnerability   `json:"Vulnerabilities"`
	ExperimentalModifiedFindings []trivyModifiedFinding `json:"ExperimentalModifiedFindings"`
}

type trivyVulnerability struct {
	VulnerabilityID string          `json:"VulnerabilityID"`
	PkgID           string          `json:"PkgID"`
	PkgName         string          `json:"PkgName"`
	PkgIdentifier   trivyIdentifier `json:"PkgIdentifier"`
	Installed       string          `json:"InstalledVersion"`
	Severity        string          `json:"Severity"`
}

type trivyIdentifier struct {
	PURL string `json:"PURL"`
}

type trivyModifiedFinding struct {
	Status  string             `json:"Status"`
	Source  string             `json:"Source"`
	Finding trivyVulnerability `json:"Finding"`
}

type cyclonedxSBOM struct {
	Metadata   cyclonedxMetadata    `json:"metadata"`
	Components []cyclonedxComponent `json:"components"`
}

type cyclonedxMetadata struct {
	Component cyclonedxComponent `json:"component"`
}

type cyclonedxComponent struct {
	Name       string              `json:"name"`
	Version    string              `json:"version"`
	PURL       string              `json:"purl"`
	Properties []cyclonedxProperty `json:"properties"`
}

type cyclonedxProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type findingContext struct {
	ResultType    string
	Vulnerability trivyVulnerability
}

type sourcePackage struct {
	Name    string
	Version string
}

type dhiLineage struct {
	Product     string
	Name        string
	Version     string
	Definition  string
	ChainID     string
	BaseDiffIDs []string
	baseLayers  map[string]struct{}
}

type projection struct {
	Statement vexStatement
	Manifest  manifestProjection
}

func Scan(ctx context.Context, opts Options) (summary Summary, err error) {
	opts = withDefaults(opts)
	expectedDigest, err := validateOptions(opts)
	if err != nil {
		return Summary{}, err
	}

	outputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return Summary{}, fmt.Errorf("resolve output directory: %w", err)
	}
	if _, err := os.Stat(outputDir); err == nil {
		return Summary{}, fmt.Errorf("output directory already exists: %s", outputDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Summary{}, fmt.Errorf("inspect output directory: %w", err)
	}
	parent := filepath.Dir(outputDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return Summary{}, fmt.Errorf("create output parent: %w", err)
	}
	stage, err := os.MkdirTemp(parent, "."+filepath.Base(outputDir)+".tmp-")
	if err != nil {
		return Summary{}, fmt.Errorf("create evidence staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(stage)
		}
	}()

	emptyConfig := filepath.Join(stage, "trivy-empty.yaml")
	emptyIgnore := filepath.Join(stage, "trivy-empty.ignore")
	if err := os.WriteFile(emptyConfig, []byte("{}\n"), 0o600); err != nil {
		return Summary{}, fmt.Errorf("write controlled Trivy config: %w", err)
	}
	if err := os.WriteFile(emptyIgnore, nil, 0o600); err != nil {
		return Summary{}, fmt.Errorf("write controlled Trivy ignore file: %w", err)
	}
	for _, directory := range []string{
		filepath.Join(stage, "home"),
		filepath.Join(stage, "xdg-cache"),
		filepath.Join(stage, "xdg-config"),
		filepath.Join(stage, "docker-config"),
		filepath.Join(stage, "trivy-cache"),
	} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return Summary{}, fmt.Errorf("create isolated command directory: %w", err)
		}
	}
	trivyCache := filepath.Join(stage, "trivy-cache")

	rawPath := filepath.Join(stage, "trivy-raw.json")
	sbomPath := filepath.Join(stage, "trivy-sbom.cdx.json")
	finalPath := filepath.Join(stage, "trivy-vex.json")
	compatPath := filepath.Join(stage, "dhi-trivy-compat.vex.json")

	logf(opts, "generating isolated raw Trivy report")
	if err := runTrivyImage(ctx, opts, stage, emptyConfig, emptyIgnore, rawPath, "", false, false); err != nil {
		return Summary{}, fmt.Errorf("generate raw Trivy report: %w", err)
	}
	logf(opts, "generating Trivy CycloneDX SBOM")
	if err := runTrivySBOM(ctx, opts, stage, emptyConfig, emptyIgnore, sbomPath); err != nil {
		return Summary{}, fmt.Errorf("generate Trivy CycloneDX SBOM: %w", err)
	}

	raw, err := readJSON[trivyReport](rawPath)
	if err != nil {
		return Summary{}, fmt.Errorf("read raw Trivy report: %w", err)
	}
	sbom, err := readJSON[cyclonedxSBOM](sbomPath)
	if err != nil {
		return Summary{}, fmt.Errorf("read Trivy SBOM: %w", err)
	}
	if err := verifyRawIdentity(raw, sbom, expectedDigest, opts.Platform); err != nil {
		return Summary{}, err
	}
	lineage, err := detectDHILineage(raw)
	if err != nil {
		return Summary{}, err
	}
	logf(
		opts,
		"verified derived DHI lineage: product=%s chain=%s base_layers=%d",
		lineage.Product,
		lineage.ChainID,
		len(lineage.BaseDiffIDs),
	)

	trivyVersionBefore, err := captureCommand(
		ctx,
		isolatedEnv(stage),
		stage,
		opts.TrivyPath,
		"version",
		"--format", "json",
		"--cache-dir", trivyCache,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("capture Trivy version: %w", err)
	}
	databaseHashesBefore, err := hashTrivyDatabase(trivyCache)
	if err != nil {
		return Summary{}, err
	}

	commit, err := resolveCommit(ctx, opts, opts.AdvisoriesRef)
	if err != nil {
		return Summary{}, err
	}
	logf(opts, "resolved Docker DHI advisories %q to %s", opts.AdvisoriesRef, commit)

	keyPath := filepath.Join(stage, "dhi-public-key.pem")
	if err := download(ctx, opts, defaultDHIKeyURL, keyPath); err != nil {
		return Summary{}, fmt.Errorf("download Docker DHI public key: %w", err)
	}
	keySHA, err := fileSHA256(keyPath)
	if err != nil {
		return Summary{}, fmt.Errorf("hash Docker DHI public key: %w", err)
	}
	if keySHA != defaultDHIKeySHA256 {
		return Summary{}, fmt.Errorf(
			"Docker DHI public key SHA-256 %s does not match trusted fingerprint %s",
			keySHA,
			defaultDHIKeySHA256,
		)
	}

	document, documentEvidence, err := fetchVerifiedProductVEX(
		ctx,
		opts,
		stage,
		commit,
		keyPath,
		lineage.Product,
	)
	if err != nil {
		return Summary{}, err
	}
	verifiedDocuments := []verifiedVEX{document}
	documentManifest := []manifestDocument{documentEvidence}

	findings := flattenFindings(raw)
	components, err := indexComponents(sbom)
	if err != nil {
		return Summary{}, err
	}
	sourceByFinding, unmatched := mapFindingSources(findings, components, lineage)
	projections, projectionUnmatched := buildProjections(findings, sourceByFinding, verifiedDocuments, commit)
	unmatched = append(unmatched, projectionUnmatched...)
	if unmatched == nil {
		unmatched = []unmatchedFinding{}
	}
	sortUnmatched(unmatched)

	generatedAt := opts.Now().UTC().Format(time.RFC3339Nano)
	compatVEX := buildCompatibilityVEX(projections, generatedAt)
	if err := writeJSON(compatPath, compatVEX); err != nil {
		return Summary{}, fmt.Errorf("write dynamic Trivy compatibility VEX: %w", err)
	}

	logf(opts, "running Trivy with the current verified Docker VEX projection")
	if err := runTrivyImage(ctx, opts, stage, emptyConfig, emptyIgnore, finalPath, compatPath, true, true); err != nil {
		return Summary{}, fmt.Errorf("run VEX-aware Trivy scan: %w", err)
	}
	final, err := readJSON[trivyReport](finalPath)
	if err != nil {
		return Summary{}, fmt.Errorf("read VEX-aware Trivy report: %w", err)
	}
	if err := verifyFinalIdentity(raw, final, expectedDigest); err != nil {
		return Summary{}, err
	}
	if err := verifyReportAccounting(raw, final); err != nil {
		return Summary{}, err
	}
	if err := verifyProjectedSuppressions(projections, final); err != nil {
		return Summary{}, err
	}
	if err := verifyUnmatchedActives(unmatched, final); err != nil {
		return Summary{}, err
	}

	trivyVersionAfter, err := captureCommand(
		ctx,
		isolatedEnv(stage),
		stage,
		opts.TrivyPath,
		"version",
		"--format", "json",
		"--cache-dir", trivyCache,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("capture final Trivy version: %w", err)
	}
	if !bytes.Equal(bytes.TrimSpace(trivyVersionBefore), bytes.TrimSpace(trivyVersionAfter)) {
		return Summary{}, errors.New("Trivy version or vulnerability database changed during the scan")
	}
	databaseHashesAfter, err := hashTrivyDatabase(trivyCache)
	if err != nil {
		return Summary{}, err
	}
	if !reflect.DeepEqual(databaseHashesBefore, databaseHashesAfter) {
		return Summary{}, errors.New("Trivy vulnerability database files changed during the scan")
	}
	versionPath := filepath.Join(stage, "trivy-version.json")
	if err := os.WriteFile(versionPath, append(bytes.TrimSpace(trivyVersionAfter), '\n'), 0o600); err != nil {
		return Summary{}, fmt.Errorf("write Trivy version evidence: %w", err)
	}

	summary = summarize(
		opts,
		expectedDigest,
		commit,
		lineage,
		len(verifiedDocuments),
		len(projections),
		len(unmatched),
		final,
	)
	reportHashes, err := hashReports(
		versionPath,
		rawPath,
		sbomPath,
		compatPath,
		finalPath,
		databaseHashesAfter,
	)
	if err != nil {
		return Summary{}, err
	}
	manifest := evidenceManifest{
		SchemaVersion:   1,
		GeneratedAt:     generatedAt,
		Image:           opts.Image,
		ImageDigest:     expectedDigest,
		PlatformImageID: raw.Metadata.ImageID,
		Lineage: manifestLineage{
			Product:     lineage.Product,
			Name:        lineage.Name,
			Version:     lineage.Version,
			Definition:  lineage.Definition,
			ChainID:     lineage.ChainID,
			BaseLayers:  len(lineage.BaseDiffIDs),
			BaseDiffIDs: append([]string(nil), lineage.BaseDiffIDs...),
		},
		Platform: opts.Platform,
		Severity: opts.Severity,
		Timeout:  opts.Timeout,
		Advisories: manifestAdvisories{
			Repository: advisoriesRepository,
			Commit:     commit,
			KeyURL:     defaultDHIKeyURL,
			KeySHA256:  keySHA,
		},
		Documents:   documentManifest,
		Projections: projectionManifestEntries(projections),
		Unmatched:   unmatched,
		Reports:     reportHashes,
	}
	sort.Slice(manifest.Documents, func(i, j int) bool {
		return manifest.Documents[i].Product < manifest.Documents[j].Product
	})
	if err := writeJSON(filepath.Join(stage, "projection-manifest.json"), manifest); err != nil {
		return Summary{}, fmt.Errorf("write projection manifest: %w", err)
	}
	if err := writeJSON(filepath.Join(stage, "summary.json"), summary); err != nil {
		return Summary{}, fmt.Errorf("write summary: %w", err)
	}
	if err := os.RemoveAll(trivyCache); err != nil {
		return Summary{}, fmt.Errorf("remove temporary Trivy cache: %w", err)
	}

	if err := os.Rename(stage, outputDir); err != nil {
		return Summary{}, fmt.Errorf("publish evidence directory atomically: %w", err)
	}
	committed = true
	summary.OutputDir = outputDir
	if err := writeJSON(filepath.Join(outputDir, "summary.json"), summary); err != nil {
		return Summary{}, fmt.Errorf("update published summary path: %w", err)
	}
	return summary, nil
}

func withDefaults(opts Options) Options {
	if opts.Platform == "" {
		opts.Platform = "linux/amd64"
	}
	if opts.Severity == "" {
		opts.Severity = "HIGH,CRITICAL"
	}
	if opts.Timeout == "" {
		opts.Timeout = "45m"
	}
	if opts.AdvisoriesRef == "" {
		opts.AdvisoriesRef = "main"
	}
	if opts.TrivyPath == "" {
		opts.TrivyPath = "trivy"
	}
	if opts.CosignPath == "" {
		opts.CosignPath = "cosign"
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 2 * time.Minute}
	}
	if opts.Log == nil {
		opts.Log = io.Discard
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return opts
}

func validateOptions(opts Options) (string, error) {
	if opts.Image == "" {
		return "", errors.New("image is required")
	}
	match := digestPattern.FindStringSubmatch(opts.Image)
	if match == nil {
		return "", errors.New("image must use an immutable @sha256 digest")
	}
	if opts.OutputDir == "" {
		return "", errors.New("output directory is required")
	}
	timeout, err := time.ParseDuration(opts.Timeout)
	if err != nil || timeout <= 0 {
		return "", fmt.Errorf("invalid Trivy timeout %q", opts.Timeout)
	}
	return "sha256:" + match[1], nil
}

func runTrivyImage(
	ctx context.Context,
	opts Options,
	workDir, configPath, ignorePath, outputPath, vexPath string,
	showSuppressed bool,
	skipDBUpdate bool,
) error {
	args := []string{
		"image",
		"--config", configPath,
		"--ignorefile", ignorePath,
		"--cache-dir", filepath.Join(workDir, "trivy-cache"),
		"--image-src", "remote",
		"--platform", opts.Platform,
		"--scanners", "vuln",
		"--severity", opts.Severity,
		"--format", "json",
		"--output", outputPath,
		"--no-progress",
		"--timeout", opts.Timeout,
	}
	if vexPath != "" {
		args = append(args, "--vex", vexPath)
	}
	if showSuppressed {
		args = append(args, "--show-suppressed")
	}
	if skipDBUpdate {
		args = append(args, "--skip-db-update", "--skip-java-db-update")
	}
	args = append(args, opts.Image)
	return runCommand(ctx, isolatedEnv(workDir), workDir, opts.Log, opts.TrivyPath, args...)
}

func runTrivySBOM(
	ctx context.Context,
	opts Options,
	workDir, configPath, ignorePath, outputPath string,
) error {
	return runCommand(
		ctx,
		isolatedEnv(workDir),
		workDir,
		opts.Log,
		opts.TrivyPath,
		"image",
		"--config", configPath,
		"--ignorefile", ignorePath,
		"--cache-dir", filepath.Join(workDir, "trivy-cache"),
		"--image-src", "remote",
		"--platform", opts.Platform,
		"--format", "cyclonedx",
		"--output", outputPath,
		"--no-progress",
		"--timeout", opts.Timeout,
		"--skip-db-update",
		"--skip-java-db-update",
		opts.Image,
	)
}

func isolatedEnv(workDir string) []string {
	allowedTrivy := map[string]bool{
		"TRIVY_USERNAME": true,
		"TRIVY_PASSWORD": true,
	}
	isolated := map[string]string{
		"HOME":            filepath.Join(workDir, "home"),
		"XDG_CACHE_HOME":  filepath.Join(workDir, "xdg-cache"),
		"XDG_CONFIG_HOME": filepath.Join(workDir, "xdg-config"),
		"DOCKER_CONFIG":   filepath.Join(workDir, "docker-config"),
	}
	result := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		key := entry
		if equals := strings.IndexByte(entry, '='); equals >= 0 {
			key = entry[:equals]
		}
		if strings.HasPrefix(key, "TRIVY_") && !allowedTrivy[key] {
			continue
		}
		if _, replace := isolated[key]; replace {
			continue
		}
		result = append(result, entry)
	}
	for key, value := range isolated {
		result = append(result, key+"="+value)
	}
	sort.Strings(result)
	return result
}

func runCommand(ctx context.Context, env []string, workDir string, log io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.Dir = workDir
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func captureCommand(ctx context.Context, env []string, workDir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func resolveCommit(ctx context.Context, opts Options, ref string) (string, error) {
	if commitPattern.MatchString(ref) {
		return ref, nil
	}
	endpoint := "https://api.github.com/repos/docker-hardened-images/advisories/commits/" + url.PathEscape(ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build Docker DHI commit request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gromit-dhi-vex")
	if opts.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+opts.GitHubToken)
	}
	resp, err := opts.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve Docker DHI advisories ref %q: %w", ref, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("resolve Docker DHI advisories ref %q: HTTP %d: %s", ref, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&response); err != nil {
		return "", fmt.Errorf("decode Docker DHI commit response: %w", err)
	}
	if !commitPattern.MatchString(response.SHA) {
		return "", fmt.Errorf("Docker DHI commit response contained invalid SHA %q", response.SHA)
	}
	return response.SHA, nil
}

func fetchVerifiedProductVEX(
	ctx context.Context,
	opts Options,
	stage, commit, keyPath, product string,
) (verifiedVEX, manifestDocument, error) {
	name := "dhi-" + product + ".vex.json"
	documentURL := rawAdvisoryURL(commit, "vex/"+product+"/"+name)
	documentPath := filepath.Join(stage, name)
	signaturePath := documentPath + ".sig"
	verificationPath := documentPath + ".cosign-verify.txt"
	if err := download(ctx, opts, documentURL, documentPath); err != nil {
		return verifiedVEX{}, manifestDocument{}, fmt.Errorf("download Docker DHI %s VEX: %w", product, err)
	}
	if err := download(ctx, opts, documentURL+".sig", signaturePath); err != nil {
		return verifiedVEX{}, manifestDocument{}, fmt.Errorf("download Docker DHI %s VEX signature: %w", product, err)
	}
	verification, err := captureCommand(
		ctx,
		isolatedEnv(stage),
		stage,
		opts.CosignPath,
		"verify-blob",
		"--bundle", signaturePath,
		"--key", keyPath,
		documentPath,
	)
	if len(verification) > 0 {
		_, _ = opts.Log.Write(verification)
	}
	if err != nil {
		return verifiedVEX{}, manifestDocument{}, fmt.Errorf("verify Docker DHI %s VEX: %w", product, err)
	}
	if err := os.WriteFile(verificationPath, append(bytes.TrimSpace(verification), '\n'), 0o600); err != nil {
		return verifiedVEX{}, manifestDocument{}, fmt.Errorf("write Docker DHI %s Cosign evidence: %w", product, err)
	}
	document, err := readJSON[vexDocument](documentPath)
	if err != nil {
		return verifiedVEX{}, manifestDocument{}, fmt.Errorf("read verified Docker DHI %s VEX: %w", product, err)
	}
	digest, err := fileSHA256(documentPath)
	if err != nil {
		return verifiedVEX{}, manifestDocument{}, fmt.Errorf("hash verified Docker DHI %s VEX: %w", product, err)
	}
	signatureDigest, err := fileSHA256(signaturePath)
	if err != nil {
		return verifiedVEX{}, manifestDocument{}, fmt.Errorf("hash verified Docker DHI %s VEX signature: %w", product, err)
	}
	verificationDigest, err := fileSHA256(verificationPath)
	if err != nil {
		return verifiedVEX{}, manifestDocument{}, fmt.Errorf("hash Docker DHI %s Cosign evidence: %w", product, err)
	}
	logf(opts, "verified current Docker DHI %s VEX", product)
	return verifiedVEX{
			Product:  product,
			URL:      documentURL,
			Document: document,
		}, manifestDocument{
			Product:            product,
			URL:                documentURL,
			SignatureURL:       documentURL + ".sig",
			SHA256:             digest,
			SignatureSHA256:    signatureDigest,
			VerificationSHA256: verificationDigest,
		}, nil
}

func download(ctx context.Context, opts Options, sourceURL, destination string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return fmt.Errorf("build request for %s: %w", sourceURL, err)
	}
	req.Header.Set("User-Agent", "gromit-dhi-vex")
	if opts.GitHubToken != "" && strings.HasPrefix(sourceURL, "https://raw.githubusercontent.com/") {
		req.Header.Set("Authorization", "Bearer "+opts.GitHubToken)
	}
	resp, err := opts.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("download %s: HTTP %d: %s", sourceURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.ContentLength > maxDownloadSize {
		return fmt.Errorf("download %s exceeds %d bytes", sourceURL, maxDownloadSize)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create %s: %w", destination, err)
	}
	written, copyErr := io.Copy(file, io.LimitReader(resp.Body, maxDownloadSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("write %s: %w", destination, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", destination, closeErr)
	}
	if written > maxDownloadSize {
		return fmt.Errorf("download %s exceeds %d bytes", sourceURL, maxDownloadSize)
	}
	return nil
}

func verifyRawIdentity(
	raw trivyReport,
	sbom cyclonedxSBOM,
	expectedDigest, platform string,
) error {
	if raw.ArtifactType != "container_image" {
		return fmt.Errorf("Trivy artifact type is %q, expected container_image", raw.ArtifactType)
	}
	if raw.Metadata.ImageID == "" {
		return errors.New("raw Trivy report has no platform image ID")
	}
	if !reportHasDigest(raw, expectedDigest) {
		return fmt.Errorf("raw Trivy report does not retain requested digest %q", expectedDigest)
	}
	if !strings.Contains(sbom.Metadata.Component.PURL, "@"+expectedDigest) {
		return fmt.Errorf("Trivy SBOM root PURL %q does not contain requested digest %q", sbom.Metadata.Component.PURL, expectedDigest)
	}
	platformParts := strings.Split(platform, "/")
	if len(platformParts) < 2 || platformParts[0] == "" || platformParts[1] == "" {
		return fmt.Errorf("invalid container platform %q", platform)
	}
	if raw.Metadata.ImageConfig.OS != platformParts[0] ||
		raw.Metadata.ImageConfig.Architecture != platformParts[1] {
		return fmt.Errorf(
			"Trivy selected image config %s/%s, requested %s",
			raw.Metadata.ImageConfig.OS,
			raw.Metadata.ImageConfig.Architecture,
			platform,
		)
	}
	return nil
}

func verifyFinalIdentity(raw, final trivyReport, expectedDigest string) error {
	if !reportHasDigest(final, expectedDigest) {
		return fmt.Errorf("final Trivy report does not retain requested digest %q", expectedDigest)
	}
	if final.Metadata.ImageID != raw.Metadata.ImageID {
		return fmt.Errorf(
			"Trivy platform image changed between scans: raw=%q final=%q",
			raw.Metadata.ImageID,
			final.Metadata.ImageID,
		)
	}
	return nil
}

func reportHasDigest(report trivyReport, expectedDigest string) bool {
	if strings.HasSuffix(report.Metadata.Reference, "@"+expectedDigest) {
		return true
	}
	for _, digest := range report.Metadata.RepoDigests {
		if strings.HasSuffix(digest, "@"+expectedDigest) {
			return true
		}
	}
	return false
}

func detectDHILineage(report trivyReport) (dhiLineage, error) {
	if report.Metadata.OS.Family != "debian" {
		return dhiLineage{}, fmt.Errorf(
			"unsupported Docker DHI operating system %q; only Debian source mapping is implemented",
			report.Metadata.OS.Family,
		)
	}
	labels := report.Metadata.ImageConfig.Config.Labels
	name := labels["com.docker.dhi.name"]
	version := labels["com.docker.dhi.version"]
	definition := labels["com.docker.dhi.definition"]
	expectedChainID := labels["com.docker.dhi.chain-id"]
	if name == "" || version == "" || definition == "" || expectedChainID == "" {
		return dhiLineage{}, errors.New(
			"image does not contain the complete Docker DHI name, version, definition, and chain-id labels",
		)
	}

	const namePrefix = "dhi/"
	if !strings.HasPrefix(name, namePrefix) || len(name) == len(namePrefix) {
		return dhiLineage{}, fmt.Errorf("unsupported Docker DHI product label %q", name)
	}
	product := strings.TrimPrefix(name, namePrefix)
	if strings.ContainsAny(product, `/\`) || product == "." || product == ".." {
		return dhiLineage{}, fmt.Errorf("invalid Docker DHI product label %q", name)
	}
	if len(report.Metadata.DiffIDs) == 0 {
		return dhiLineage{}, errors.New("Trivy report has no layer DiffIDs for Docker DHI lineage verification")
	}

	baseLayers := make(map[string]struct{})
	baseDiffIDs := make([]string, 0, len(report.Metadata.DiffIDs))
	chainID := ""
	found := false
	for _, diffID := range report.Metadata.DiffIDs {
		if !strings.HasPrefix(diffID, "sha256:") {
			return dhiLineage{}, fmt.Errorf("unsupported layer DiffID %q", diffID)
		}
		if chainID == "" {
			chainID = diffID
		} else {
			chainID = "sha256:" + sha256Hex(chainID+" "+diffID)
		}
		baseDiffIDs = append(baseDiffIDs, diffID)
		baseLayers[diffID] = struct{}{}
		if chainID == expectedChainID {
			found = true
			break
		}
	}
	if !found {
		return dhiLineage{}, fmt.Errorf(
			"Docker DHI chain-id %q does not match any target image layer boundary",
			expectedChainID,
		)
	}

	return dhiLineage{
		Product:     product,
		Name:        name,
		Version:     version,
		Definition:  definition,
		ChainID:     expectedChainID,
		BaseDiffIDs: baseDiffIDs,
		baseLayers:  baseLayers,
	}, nil
}

func flattenFindings(report trivyReport) []findingContext {
	var findings []findingContext
	for _, result := range report.Results {
		for _, vulnerability := range result.Vulnerabilities {
			findings = append(findings, findingContext{
				ResultType:    result.Type,
				Vulnerability: vulnerability,
			})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		left := findings[i].Vulnerability
		right := findings[j].Vulnerability
		if left.PkgIdentifier.PURL != right.PkgIdentifier.PURL {
			return left.PkgIdentifier.PURL < right.PkgIdentifier.PURL
		}
		return left.VulnerabilityID < right.VulnerabilityID
	})
	return findings
}

func indexComponents(sbom cyclonedxSBOM) (map[string]cyclonedxComponent, error) {
	components := make(map[string]cyclonedxComponent, len(sbom.Components))
	for _, component := range sbom.Components {
		if component.PURL == "" {
			continue
		}
		if existing, ok := components[component.PURL]; ok {
			if existing.Name != component.Name || existing.Version != component.Version ||
				!equalComponentMetadata(existing, component) {
				return nil, fmt.Errorf("Trivy SBOM contains conflicting components for PURL %q", component.PURL)
			}
			continue
		}
		components[component.PURL] = component
	}
	return components, nil
}

func equalComponentMetadata(left, right cyclonedxComponent) bool {
	leftSource, leftErr := sourceFromComponent(left)
	rightSource, rightErr := sourceFromComponent(right)
	if (leftErr == nil) != (rightErr == nil) || leftSource != rightSource {
		return false
	}
	leftLayer, leftLayerErr := componentProperty(left, "aquasecurity:trivy:LayerDiffID")
	rightLayer, rightLayerErr := componentProperty(right, "aquasecurity:trivy:LayerDiffID")
	return (leftLayerErr == nil) == (rightLayerErr == nil) && leftLayer == rightLayer
}

func mapFindingSources(
	findings []findingContext,
	components map[string]cyclonedxComponent,
	lineage dhiLineage,
) (map[string]sourcePackage, []unmatchedFinding) {
	result := make(map[string]sourcePackage)
	var unmatched []unmatchedFinding
	for _, finding := range findings {
		vulnerability := finding.Vulnerability
		key := findingKey(vulnerability)
		if finding.ResultType != "debian" || !strings.HasPrefix(vulnerability.PkgIdentifier.PURL, "pkg:deb/debian/") {
			unmatched = append(unmatched, newUnmatched(vulnerability, "finding is not a Debian OS package"))
			continue
		}
		component, ok := components[vulnerability.PkgIdentifier.PURL]
		if !ok {
			unmatched = append(unmatched, newUnmatched(vulnerability, "exact package PURL is absent from the Trivy SBOM"))
			continue
		}
		layerDiffID, err := componentProperty(component, "aquasecurity:trivy:LayerDiffID")
		if err != nil {
			unmatched = append(unmatched, newUnmatched(vulnerability, err.Error()))
			continue
		}
		if _, inherited := lineage.baseLayers[layerDiffID]; !inherited {
			unmatched = append(unmatched, newUnmatched(
				vulnerability,
				fmt.Sprintf("package layer %s is outside the verified Docker DHI base boundary", layerDiffID),
			))
			continue
		}
		source, err := sourceFromComponent(component)
		if err != nil {
			unmatched = append(unmatched, newUnmatched(vulnerability, err.Error()))
			continue
		}
		result[key] = source
	}
	return result, unmatched
}

func componentProperty(component cyclonedxComponent, name string) (string, error) {
	var value string
	for _, property := range component.Properties {
		if property.Name != name {
			continue
		}
		if value != "" && value != property.Value {
			return "", fmt.Errorf("conflicting %s values for %q", name, component.PURL)
		}
		value = property.Value
	}
	if value == "" {
		return "", fmt.Errorf("missing %s for %q", name, component.PURL)
	}
	return value, nil
}

func sourceFromComponent(component cyclonedxComponent) (sourcePackage, error) {
	properties := make(map[string]string, len(component.Properties))
	for _, property := range component.Properties {
		if existing, ok := properties[property.Name]; ok && existing != property.Value {
			return sourcePackage{}, fmt.Errorf(
				"conflicting %s values for %q",
				property.Name,
				component.PURL,
			)
		}
		properties[property.Name] = property.Value
	}
	const prefix = "aquasecurity:trivy:"
	name := properties[prefix+"SrcName"]
	version := properties[prefix+"SrcVersion"]
	release := properties[prefix+"SrcRelease"]
	epoch := properties[prefix+"SrcEpoch"]
	if name == "" || version == "" {
		return sourcePackage{}, fmt.Errorf("incomplete Trivy source metadata for %q", component.PURL)
	}
	if epoch != "" {
		version = epoch + ":" + version
	}
	if release != "" {
		version += "-" + release
	}
	return sourcePackage{Name: name, Version: version}, nil
}

func buildProjections(
	findings []findingContext,
	sourceByFinding map[string]sourcePackage,
	documents []verifiedVEX,
	commit string,
) ([]projection, []unmatchedFinding) {
	var projections []projection
	var unmatched []unmatchedFinding
	for _, finding := range findings {
		vulnerability := finding.Vulnerability
		source, ok := sourceByFinding[findingKey(vulnerability)]
		if !ok {
			continue
		}
		statement, sourcePURL, documentURL, reason, ok := selectCurrentNotAffected(
			documents,
			vulnerability.VulnerabilityID,
			source,
		)
		if !ok {
			unmatched = append(unmatched, newUnmatched(vulnerability, reason))
			continue
		}
		id := sha256Hex(strings.Join([]string{
			commit,
			documentURL,
			statement.ID,
			vulnerability.VulnerabilityID,
			vulnerability.PkgIdentifier.PURL,
		}, "\x00"))
		projected := statement
		projected.ID = "https://tyk.io/vex/dhi-trivy-interoperability/statement/" + id
		projected.Products = []vexProduct{{ID: vulnerability.PkgIdentifier.PURL}}
		note := "Dynamic interoperability projection from the current Cosign-verified Docker DHI VEX; only the exact Trivy binary-package PURL was added."
		if projected.StatusNotes == "" {
			projected.StatusNotes = note
		} else {
			projected.StatusNotes += "\n" + note
		}
		projections = append(projections, projection{
			Statement: projected,
			Manifest: manifestProjection{
				VulnerabilityID: vulnerability.VulnerabilityID,
				BinaryPackage:   vulnerability.PkgName,
				BinaryVersion:   vulnerability.Installed,
				BinaryPURL:      vulnerability.PkgIdentifier.PURL,
				SourcePackage:   source.Name,
				SourceVersion:   source.Version,
				SourcePURL:      sourcePURL,
				SourceDocument:  documentURL,
				SourceStatement: statement.ID,
				Status:          statement.Status,
				Justification:   statement.Justification,
				StatementTime:   statement.Timestamp,
			},
		})
	}
	sort.Slice(projections, func(i, j int) bool {
		left := projections[i].Manifest
		right := projections[j].Manifest
		if left.BinaryPURL != right.BinaryPURL {
			return left.BinaryPURL < right.BinaryPURL
		}
		return left.VulnerabilityID < right.VulnerabilityID
	})
	return projections, unmatched
}

func selectCurrentNotAffected(
	documents []verifiedVEX,
	vulnerabilityID string,
	source sourcePackage,
) (vexStatement, string, string, string, bool) {
	type candidate struct {
		statement vexStatement
		product   string
		document  string
		time      time.Time
	}
	var candidates []candidate
	for _, document := range documents {
		for _, statement := range document.Document.Statements {
			if statement.Vulnerability.Name != vulnerabilityID {
				continue
			}
			var exactProduct string
			for _, product := range statement.Products {
				name, version, ok := parseVersionedDebianPURL(product.ID)
				if !ok || name != source.Name || version != source.Version {
					continue
				}
				if exactProduct == "" || len(product.ID) < len(exactProduct) {
					exactProduct = product.ID
				}
			}
			if exactProduct == "" {
				continue
			}
			statementTime, err := time.Parse(time.RFC3339Nano, statement.Timestamp)
			if err != nil {
				return vexStatement{}, "", "", fmt.Sprintf(
					"Docker VEX statement for %s and %s has invalid timestamp %q",
					vulnerabilityID,
					source.Name,
					statement.Timestamp,
				), false
			}
			candidates = append(candidates, candidate{
				statement: statement,
				product:   exactProduct,
				document:  document.URL,
				time:      statementTime,
			})
		}
	}
	if len(candidates) == 0 {
		return vexStatement{}, "", "", fmt.Sprintf(
			"current Docker VEX has no exact statement for %s and %s@%s",
			vulnerabilityID,
			source.Name,
			source.Version,
		), false
	}
	latest := candidates[0].time
	for _, candidate := range candidates[1:] {
		if candidate.time.After(latest) {
			latest = candidate.time
		}
	}
	var current []candidate
	for _, candidate := range candidates {
		if candidate.time.Equal(latest) {
			current = append(current, candidate)
		}
	}
	for _, candidate := range current {
		if candidate.statement.Status != "not_affected" {
			return vexStatement{}, "", "", fmt.Sprintf(
				"current exact Docker VEX status for %s and %s@%s is %s",
				vulnerabilityID,
				source.Name,
				source.Version,
				candidate.statement.Status,
			), false
		}
	}
	sort.Slice(current, func(i, j int) bool {
		if current[i].document != current[j].document {
			return current[i].document < current[j].document
		}
		return current[i].product < current[j].product
	})
	return current[0].statement, current[0].product, current[0].document, "", true
}

func parseVersionedDebianPURL(purl string) (string, string, bool) {
	const prefix = "pkg:deb/debian/"
	if !strings.HasPrefix(purl, prefix) {
		return "", "", false
	}
	value := strings.TrimPrefix(purl, prefix)
	if question := strings.IndexByte(value, '?'); question >= 0 {
		value = value[:question]
	}
	at := strings.LastIndexByte(value, '@')
	if at <= 0 || at == len(value)-1 {
		return "", "", false
	}
	name, err := url.PathUnescape(value[:at])
	if err != nil {
		return "", "", false
	}
	version, err := url.PathUnescape(value[at+1:])
	if err != nil {
		return "", "", false
	}
	return name, version, true
}

func buildCompatibilityVEX(projections []projection, timestamp string) vexDocument {
	statements := make([]vexStatement, 0, len(projections))
	hash := sha256.New()
	for _, projection := range projections {
		statements = append(statements, projection.Statement)
		_, _ = io.WriteString(hash, projection.Statement.ID)
		_, _ = io.WriteString(hash, "\x00")
	}
	return vexDocument{
		Context:    "https://openvex.dev/ns/v0.2.0",
		ID:         "https://tyk.io/vex/dhi-trivy-interoperability/" + hex.EncodeToString(hash.Sum(nil)),
		Author:     "Tyk Technologies <security@tyk.io>",
		Role:       "Document Translator",
		Version:    1,
		Timestamp:  timestamp,
		Statements: statements,
	}
}

func verifyReportAccounting(raw, final trivyReport) error {
	rawCounts := reportFindingCounts(raw)
	finalCounts := reportFindingCounts(final)
	for _, result := range final.Results {
		for _, finding := range result.ExperimentalModifiedFindings {
			finalCounts[findingAccountingKey(result.Type, finding.Finding)]++
		}
	}
	if len(rawCounts) != len(finalCounts) {
		return fmt.Errorf("Trivy findings changed between scans: raw=%d final=%d unique findings", len(rawCounts), len(finalCounts))
	}
	for key, count := range rawCounts {
		if finalCounts[key] != count {
			return fmt.Errorf("Trivy finding changed between scans for %q: raw=%d final=%d", key, count, finalCounts[key])
		}
	}
	return nil
}

func reportFindingCounts(report trivyReport) map[string]int {
	result := make(map[string]int)
	for _, scanResult := range report.Results {
		for _, vulnerability := range scanResult.Vulnerabilities {
			result[findingAccountingKey(scanResult.Type, vulnerability)]++
		}
	}
	return result
}

func verifyProjectedSuppressions(projections []projection, report trivyReport) error {
	expected := make(map[string]int)
	for _, projection := range projections {
		key := projection.Manifest.BinaryPURL + "\x00" + projection.Manifest.VulnerabilityID
		expected[key]++
	}
	suppressed := make(map[string]int)
	for _, result := range report.Results {
		for _, finding := range result.ExperimentalModifiedFindings {
			if finding.Status != "not_affected" {
				return fmt.Errorf(
					"Trivy emitted unexpected modified status %q for %s",
					finding.Status,
					findingKey(finding.Finding),
				)
			}
			suppressed[findingKey(finding.Finding)]++
		}
	}
	var missing []string
	for key, count := range expected {
		if suppressed[key] != count {
			missing = append(missing, fmt.Sprintf("%s expected=%d suppressed=%d", key, count, suppressed[key]))
		}
	}
	var unexpected []string
	for key, count := range suppressed {
		if expected[key] != count {
			unexpected = append(unexpected, fmt.Sprintf("%s projected=%d suppressed=%d", key, expected[key], count))
		}
	}
	if len(missing) > 0 || len(unexpected) > 0 {
		sort.Strings(missing)
		sort.Strings(unexpected)
		return fmt.Errorf(
			"Trivy suppression set does not exactly match verified projections: missing_or_changed=[%s] unexpected_or_changed=[%s]",
			strings.Join(missing, ", "),
			strings.Join(unexpected, ", "),
		)
	}
	return nil
}

func verifyUnmatchedActives(unmatched []unmatchedFinding, report trivyReport) error {
	expected := make(map[string]int)
	for _, finding := range unmatched {
		expected[finding.PURL+"\x00"+finding.VulnerabilityID]++
	}
	active := make(map[string]int)
	for _, result := range report.Results {
		for _, finding := range result.Vulnerabilities {
			active[findingKey(finding)]++
		}
	}
	if reflect.DeepEqual(expected, active) {
		return nil
	}
	return fmt.Errorf(
		"active Trivy findings do not exactly match the unmatched ledger: unmatched=%d active=%d unique findings",
		len(expected),
		len(active),
	)
}

func summarize(
	opts Options,
	digest, commit string,
	lineage dhiLineage,
	verifiedDocuments, projectedFindings, unmatchedFindings int,
	report trivyReport,
) Summary {
	summary := Summary{
		Image:             opts.Image,
		ImageDigest:       digest,
		PlatformImageID:   report.Metadata.ImageID,
		DHIProduct:        lineage.Name,
		DHIChainID:        lineage.ChainID,
		DHIBaseLayers:     len(lineage.BaseDiffIDs),
		Platform:          opts.Platform,
		Severity:          opts.Severity,
		AdvisoriesCommit:  commit,
		VerifiedDocuments: verifiedDocuments,
		ProjectedFindings: projectedFindings,
		UnmatchedFindings: unmatchedFindings,
		OutputDir:         opts.OutputDir,
	}
	for _, result := range report.Results {
		for _, vulnerability := range result.Vulnerabilities {
			summary.ActiveFindings++
			switch vulnerability.Severity {
			case "CRITICAL":
				summary.ActiveCritical++
			case "HIGH":
				summary.ActiveHigh++
			}
		}
		for _, finding := range result.ExperimentalModifiedFindings {
			summary.SuppressedFindings++
			switch finding.Finding.Severity {
			case "CRITICAL":
				summary.SuppressedCritical++
			case "HIGH":
				summary.SuppressedHigh++
			}
		}
	}
	return summary
}

func hashReports(
	versionPath, rawPath, sbomPath, compatPath, finalPath string,
	databaseHashes map[string]string,
) (manifestReports, error) {
	paths := []string{versionPath, rawPath, sbomPath, compatPath, finalPath}
	hashes := make([]string, 0, len(paths))
	for _, path := range paths {
		digest, err := fileSHA256(path)
		if err != nil {
			return manifestReports{}, fmt.Errorf("hash evidence file %s: %w", path, err)
		}
		hashes = append(hashes, digest)
	}
	return manifestReports{
		TrivyVersionSHA256:  hashes[0],
		DatabaseFilesSHA256: databaseHashes,
		RawSHA256:           hashes[1],
		SBOMSHA256:          hashes[2],
		CompatibilityVEX:    hashes[3],
		FinalSHA256:         hashes[4],
	}, nil
}

func hashTrivyDatabase(cacheDir string) (map[string]string, error) {
	required := []string{
		filepath.Join("db", "metadata.json"),
		filepath.Join("db", "trivy.db"),
	}
	optional := []string{
		filepath.Join("java-db", "metadata.json"),
		filepath.Join("java-db", "trivy-java.db"),
	}
	hashes := make(map[string]string, len(required)+len(optional))
	for _, relative := range required {
		path := filepath.Join(cacheDir, relative)
		digest, err := fileSHA256(path)
		if err != nil {
			return nil, fmt.Errorf("hash Trivy vulnerability database file %s: %w", relative, err)
		}
		hashes[filepath.ToSlash(relative)] = digest
	}
	for _, relative := range optional {
		path := filepath.Join(cacheDir, relative)
		digest, err := fileSHA256(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("hash Trivy vulnerability database file %s: %w", relative, err)
		}
		hashes[filepath.ToSlash(relative)] = digest
	}
	return hashes, nil
}

func projectionManifestEntries(projections []projection) []manifestProjection {
	result := make([]manifestProjection, 0, len(projections))
	for _, projection := range projections {
		result = append(result, projection.Manifest)
	}
	return result
}

func rawAdvisoryURL(commit, location string) string {
	return "https://raw.githubusercontent.com/docker-hardened-images/advisories/" + commit + "/" + location
}

func findingKey(vulnerability trivyVulnerability) string {
	return vulnerability.PkgIdentifier.PURL + "\x00" + vulnerability.VulnerabilityID
}

func findingAccountingKey(resultType string, vulnerability trivyVulnerability) string {
	return strings.Join([]string{
		resultType,
		vulnerability.PkgIdentifier.PURL,
		vulnerability.VulnerabilityID,
		vulnerability.PkgName,
		vulnerability.PkgID,
		vulnerability.Installed,
		vulnerability.Severity,
	}, "\x00")
}

func newUnmatched(vulnerability trivyVulnerability, reason string) unmatchedFinding {
	return unmatchedFinding{
		VulnerabilityID: vulnerability.VulnerabilityID,
		Package:         vulnerability.PkgName,
		PURL:            vulnerability.PkgIdentifier.PURL,
		Reason:          reason,
	}
}

func sortUnmatched(unmatched []unmatchedFinding) {
	sort.Slice(unmatched, func(i, j int) bool {
		if unmatched[i].PURL != unmatched[j].PURL {
			return unmatched[i].PURL < unmatched[j].PURL
		}
		return unmatched[i].VulnerabilityID < unmatched[j].VulnerabilityID
	})
}

func readJSON[T any](path string) (T, error) {
	var result T
	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&result); err != nil {
		return result, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return result, errors.New("JSON file contains multiple values")
		}
		return result, fmt.Errorf("read trailing JSON data: %w", err)
	}
	return result, nil
}

func writeJSON(path string, value any) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(value)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func logf(opts Options, format string, args ...any) {
	_, _ = fmt.Fprintf(opts.Log, format+"\n", args...)
}
