package dhivex

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateOptionsRequiresImmutableDigest(t *testing.T) {
	t.Parallel()

	_, err := validateOptions(withDefaults(Options{
		Image:     "tykio/tyk-plugin-compiler-fips:v5.15.0-alpha8-ng",
		OutputDir: "report",
	}))
	if err == nil || !strings.Contains(err.Error(), "immutable @sha256 digest") {
		t.Fatalf("validateOptions() error = %v, want immutable digest error", err)
	}
}

func TestSourceFromComponentUsesExactDebianSourceVersion(t *testing.T) {
	t.Parallel()

	source, err := sourceFromComponent(cyclonedxComponent{
		Name:    "linux-libc-dev",
		Version: "6.12.38-1",
		PURL:    "pkg:deb/debian/linux-libc-dev@6.12.38-1",
		Properties: []cyclonedxProperty{
			{Name: "aquasecurity:trivy:SrcName", Value: "linux"},
			{Name: "aquasecurity:trivy:SrcEpoch", Value: "1"},
			{Name: "aquasecurity:trivy:SrcVersion", Value: "6.12.38"},
			{Name: "aquasecurity:trivy:SrcRelease", Value: "1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := (sourcePackage{Name: "linux", Version: "1:6.12.38-1"}); source != want {
		t.Fatalf("sourceFromComponent() = %#v, want %#v", source, want)
	}
}

func TestSelectCurrentNotAffectedRequiresExactVersionedStatement(t *testing.T) {
	t.Parallel()

	document := verifiedVEX{
		URL: "https://example.test/dhi-busybox.vex.json",
		Document: vexDocument{Statements: []vexStatement{
			{
				ID:            "versionless",
				Vulnerability: vexVulnerability{Name: "CVE-2026-0001"},
				Products:      []vexProduct{{ID: "pkg:deb/debian/busybox"}},
				Status:        "not_affected",
				Timestamp:     "2026-07-27T00:00:00Z",
			},
			{
				ID:            "wrong-version",
				Vulnerability: vexVulnerability{Name: "CVE-2026-0001"},
				Products:      []vexProduct{{ID: "pkg:deb/debian/busybox@1.36.1-1"}},
				Status:        "not_affected",
				Timestamp:     "2026-07-27T00:00:00Z",
			},
		}},
	}

	_, _, _, reason, ok := selectCurrentNotAffected(
		[]verifiedVEX{document},
		"CVE-2026-0001",
		sourcePackage{Name: "busybox", Version: "1:1.37.0-6"},
	)
	if ok {
		t.Fatal("selectCurrentNotAffected() accepted a broad or mismatched statement")
	}
	if !strings.Contains(reason, "no exact statement") {
		t.Fatalf("reason = %q, want exact-statement failure", reason)
	}
}

func TestSelectCurrentNotAffectedRejectsNewerAffectedDecision(t *testing.T) {
	t.Parallel()

	product := "pkg:deb/debian/linux@6.12.38-1"
	document := verifiedVEX{
		URL: "https://example.test/dhi-golang.vex.json",
		Document: vexDocument{Statements: []vexStatement{
			{
				ID:            "old-not-affected",
				Vulnerability: vexVulnerability{Name: "CVE-2026-0002"},
				Products:      []vexProduct{{ID: product}},
				Status:        "not_affected",
				Timestamp:     "2026-07-26T00:00:00Z",
			},
			{
				ID:            "new-under-investigation",
				Vulnerability: vexVulnerability{Name: "CVE-2026-0002"},
				Products:      []vexProduct{{ID: product}},
				Status:        "under_investigation",
				Timestamp:     "2026-07-27T00:00:00Z",
			},
		}},
	}

	_, _, _, reason, ok := selectCurrentNotAffected(
		[]verifiedVEX{document},
		"CVE-2026-0002",
		sourcePackage{Name: "linux", Version: "6.12.38-1"},
	)
	if ok {
		t.Fatal("selectCurrentNotAffected() accepted an obsolete not_affected decision")
	}
	if !strings.Contains(reason, "under_investigation") {
		t.Fatalf("reason = %q, want current status", reason)
	}
}

func TestSelectCurrentNotAffectedAcceptsCurrentExactDecision(t *testing.T) {
	t.Parallel()

	statement := vexStatement{
		ID:            "current-not-affected",
		Vulnerability: vexVulnerability{Name: "CVE-2026-0003"},
		Products:      []vexProduct{{ID: "pkg:deb/debian/busybox@1%3A1.37.0-6?os_distro=trixie"}},
		Status:        "not_affected",
		Justification: "vulnerable_code_not_in_execute_path",
		Timestamp:     "2026-07-27T00:00:00Z",
	}
	got, purl, _, reason, ok := selectCurrentNotAffected(
		[]verifiedVEX{{
			URL:      "https://example.test/dhi-busybox.vex.json",
			Document: vexDocument{Statements: []vexStatement{statement}},
		}},
		"CVE-2026-0003",
		sourcePackage{Name: "busybox", Version: "1:1.37.0-6"},
	)
	if !ok {
		t.Fatalf("selectCurrentNotAffected() failed: %s", reason)
	}
	if got.ID != statement.ID || purl != statement.Products[0].ID {
		t.Fatalf("selectCurrentNotAffected() = (%q, %q), want (%q, %q)", got.ID, purl, statement.ID, statement.Products[0].ID)
	}
}

func TestSanitizedEnvRemovesTrivyPolicyOverrides(t *testing.T) {
	t.Setenv("TRIVY_IGNORE_UNFIXED", "true")
	t.Setenv("TRIVY_SEVERITY", "LOW")
	t.Setenv("TRIVY_USERNAME", "registry-user")
	t.Setenv("TRIVY_PASSWORD", "registry-password")

	workDir := t.TempDir()
	env := isolatedEnv(workDir)
	assertEnvironmentValue(t, env, "TRIVY_USERNAME", "registry-user")
	assertEnvironmentValue(t, env, "TRIVY_PASSWORD", "registry-password")
	assertEnvironmentMissing(t, env, "TRIVY_IGNORE_UNFIXED")
	assertEnvironmentMissing(t, env, "TRIVY_SEVERITY")
	assertEnvironmentValue(t, env, "HOME", filepath.Join(workDir, "home"))
	assertEnvironmentValue(t, env, "DOCKER_CONFIG", filepath.Join(workDir, "docker-config"))
}

func TestReportAccountingIncludesSuppressedFindings(t *testing.T) {
	t.Parallel()

	finding := trivyVulnerability{
		VulnerabilityID: "CVE-2026-0004",
		PkgIdentifier:   trivyIdentifier{PURL: "pkg:deb/debian/busybox@1.37.0-6"},
	}
	raw := trivyReport{
		Results: []trivyResult{{Vulnerabilities: []trivyVulnerability{finding}}},
	}
	final := trivyReport{
		Results: []trivyResult{{
			ExperimentalModifiedFindings: []trivyModifiedFinding{{
				Status:  "not_affected",
				Finding: finding,
			}},
		}},
	}
	if err := verifyReportAccounting(raw, final); err != nil {
		t.Fatal(err)
	}
}

func TestReportAccountingRejectsDroppedFinding(t *testing.T) {
	t.Parallel()

	raw := trivyReport{
		Results: []trivyResult{{Vulnerabilities: []trivyVulnerability{{
			VulnerabilityID: "CVE-2026-0005",
			PkgIdentifier:   trivyIdentifier{PURL: "pkg:deb/debian/busybox@1.37.0-6"},
		}}}},
	}
	if err := verifyReportAccounting(raw, trivyReport{}); err == nil {
		t.Fatal("verifyReportAccounting() accepted a dropped finding")
	}
}

func TestVerifyProjectedSuppressionsRejectsSideEffect(t *testing.T) {
	t.Parallel()

	projected := projection{
		Manifest: manifestProjection{
			VulnerabilityID: "CVE-2026-0008",
			BinaryPackage:   "busybox",
			BinaryPURL:      "pkg:deb/debian/busybox@1.37.0-6",
		},
	}
	report := trivyReport{
		Results: []trivyResult{{
			ExperimentalModifiedFindings: []trivyModifiedFinding{
				{
					Status: "not_affected",
					Finding: trivyVulnerability{
						VulnerabilityID: "CVE-2026-0008",
						PkgIdentifier: trivyIdentifier{
							PURL: "pkg:deb/debian/busybox@1.37.0-6",
						},
					},
				},
				{
					Status: "not_affected",
					Finding: trivyVulnerability{
						VulnerabilityID: "CVE-2026-0009",
						PkgIdentifier: trivyIdentifier{
							PURL: "pkg:deb/debian/gpgv@2.4.7-21",
						},
					},
				},
			},
		}},
	}
	if err := verifyProjectedSuppressions([]projection{projected}, report); err == nil {
		t.Fatal("verifyProjectedSuppressions() accepted an unprojected suppression")
	}
}

func TestVerifyUnmatchedActivesRequiresExactLedger(t *testing.T) {
	t.Parallel()

	finding := trivyVulnerability{
		VulnerabilityID: "CVE-2026-0010",
		PkgIdentifier: trivyIdentifier{
			PURL: "pkg:deb/debian/linux-libc-dev@6.12.96-1%2Bdhi0",
		},
	}
	report := trivyReport{
		Results: []trivyResult{{Vulnerabilities: []trivyVulnerability{finding}}},
	}
	unmatched := []unmatchedFinding{{
		VulnerabilityID: finding.VulnerabilityID,
		PURL:            finding.PkgIdentifier.PURL,
		Reason:          "outside the verified Docker DHI base boundary",
	}}
	if err := verifyUnmatchedActives(unmatched, report); err != nil {
		t.Fatal(err)
	}
	if err := verifyUnmatchedActives(nil, report); err == nil {
		t.Fatal("verifyUnmatchedActives() accepted an active finding absent from the ledger")
	}
}

func TestVerifyIdentityAcceptsIndexDigestAndStablePlatformImage(t *testing.T) {
	t.Parallel()

	const (
		indexDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		imageID     = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	raw := trivyReport{
		ArtifactType: "container_image",
		Metadata: trivyMetadata{
			ImageID:     imageID,
			RepoDigests: []string{"example.test/image@" + indexDigest},
			Reference:   "example.test/image@" + indexDigest,
			ImageConfig: trivyImageConfig{OS: "linux", Architecture: "amd64"},
		},
	}
	sbom := cyclonedxSBOM{
		Metadata: cyclonedxMetadata{
			Component: cyclonedxComponent{PURL: "pkg:oci/image@" + indexDigest + "?arch=amd64"},
		},
	}
	if err := verifyRawIdentity(raw, sbom, indexDigest, "linux/amd64"); err != nil {
		t.Fatal(err)
	}
	if err := verifyFinalIdentity(raw, raw, indexDigest); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyFinalIdentityRejectsChangedPlatformImage(t *testing.T) {
	t.Parallel()

	const indexDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	raw := trivyReport{
		Metadata: trivyMetadata{
			ImageID:     "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			RepoDigests: []string{"example.test/image@" + indexDigest},
		},
	}
	final := raw
	final.Metadata.ImageID = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if err := verifyFinalIdentity(raw, final, indexDigest); err == nil {
		t.Fatal("verifyFinalIdentity() accepted a changed platform image")
	}
}

func TestDetectDHILineageFindsSignedBaseBoundary(t *testing.T) {
	t.Parallel()

	const (
		firstDiffID  = "sha256:e6658c58abb0371bc49b64fa0b02994d00bc7e2e37d71dd6f435084d8d7e9e17"
		secondDiffID = "sha256:bb8c9f2e3ea36bc37b0282b477025f859e06c38c967fde95d8a2ad864204be61"
		childDiffID  = "sha256:23da6fe0d694f027f3db2d83a38a3b05d50326534cd2b8a0b3859aa60cebd728"
		chainID      = "sha256:61c42d0892b6700cdc76bb85bd17b95b08fc9df28198d9260f94828e67754d9d"
	)
	report := trivyReport{
		Metadata: trivyMetadata{
			DiffIDs: []string{firstDiffID, secondDiffID, childDiffID},
			OS:      trivyOS{Family: "debian", Name: "13.6"},
			ImageConfig: trivyImageConfig{
				Config: trivyContainerConfig{Labels: map[string]string{
					"com.docker.dhi.name":       "dhi/busybox",
					"com.docker.dhi.version":    "1.37.0-debian13-fips",
					"com.docker.dhi.definition": "image/busybox/debian-13/1-fips",
					"com.docker.dhi.chain-id":   chainID,
				}},
			},
		},
	}

	lineage, err := detectDHILineage(report)
	if err != nil {
		t.Fatal(err)
	}
	if lineage.Product != "busybox" || len(lineage.BaseDiffIDs) != 2 {
		t.Fatalf("detectDHILineage() = product %q, %d layers", lineage.Product, len(lineage.BaseDiffIDs))
	}
	if _, ok := lineage.baseLayers[childDiffID]; ok {
		t.Fatal("child layer was included inside the DHI base boundary")
	}
}

func TestDetectDHILineageRejectsUnmatchedChainID(t *testing.T) {
	t.Parallel()

	report := trivyReport{
		Metadata: trivyMetadata{
			DiffIDs: []string{
				"sha256:e6658c58abb0371bc49b64fa0b02994d00bc7e2e37d71dd6f435084d8d7e9e17",
			},
			OS: trivyOS{Family: "debian", Name: "13.6"},
			ImageConfig: trivyImageConfig{
				Config: trivyContainerConfig{Labels: map[string]string{
					"com.docker.dhi.name":       "dhi/busybox",
					"com.docker.dhi.version":    "1.37.0-debian13-fips",
					"com.docker.dhi.definition": "image/busybox/debian-13/1-fips",
					"com.docker.dhi.chain-id":   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				}},
			},
		},
	}
	if _, err := detectDHILineage(report); err == nil {
		t.Fatal("detectDHILineage() accepted an unmatched chain-id")
	}
}

func TestMapFindingSourcesRejectsCustomerLayerPackage(t *testing.T) {
	t.Parallel()

	const (
		purl       = "pkg:deb/debian/linux-libc-dev@6.12.96-1%2Bdhi0"
		baseLayer  = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		childLayer = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	finding := findingContext{
		ResultType: "debian",
		Vulnerability: trivyVulnerability{
			VulnerabilityID: "CVE-2026-0007",
			PkgName:         "linux-libc-dev",
			PkgIdentifier:   trivyIdentifier{PURL: purl},
		},
	}
	component := cyclonedxComponent{
		Name:    "linux-libc-dev",
		Version: "6.12.96-1+dhi0",
		PURL:    purl,
		Properties: []cyclonedxProperty{
			{Name: "aquasecurity:trivy:LayerDiffID", Value: childLayer},
			{Name: "aquasecurity:trivy:SrcName", Value: "linux"},
			{Name: "aquasecurity:trivy:SrcVersion", Value: "6.12.96"},
			{Name: "aquasecurity:trivy:SrcRelease", Value: "1+dhi0"},
		},
	}
	sources, unmatched := mapFindingSources(
		[]findingContext{finding},
		map[string]cyclonedxComponent{purl: component},
		dhiLineage{baseLayers: map[string]struct{}{baseLayer: {}}},
	)
	if len(sources) != 0 || len(unmatched) != 1 {
		t.Fatalf("mapFindingSources() = %d sources, %d unmatched", len(sources), len(unmatched))
	}
	if !strings.Contains(unmatched[0].Reason, "outside the verified Docker DHI base boundary") {
		t.Fatalf("unexpected unmatched reason: %q", unmatched[0].Reason)
	}
}

func TestIndexComponentsRejectsConflictingLayerAttribution(t *testing.T) {
	t.Parallel()

	const purl = "pkg:deb/debian/busybox@1.37.0-6%2Bdhi1"
	component := cyclonedxComponent{
		Name:    "busybox",
		Version: "1.37.0-6+dhi1",
		PURL:    purl,
		Properties: []cyclonedxProperty{{
			Name:  "aquasecurity:trivy:LayerDiffID",
			Value: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}},
	}
	conflict := component
	conflict.Properties = []cyclonedxProperty{{
		Name:  "aquasecurity:trivy:LayerDiffID",
		Value: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}}
	if _, err := indexComponents(cyclonedxSBOM{
		Components: []cyclonedxComponent{component, conflict},
	}); err == nil {
		t.Fatal("indexComponents() accepted conflicting layer attribution")
	}
}

func TestBuildCompatibilityVEXPreservesDockerDecision(t *testing.T) {
	t.Parallel()

	statement := vexStatement{
		ID:            "docker-statement",
		Vulnerability: vexVulnerability{Name: "CVE-2026-0006"},
		Status:        "not_affected",
		Justification: "component_not_present",
		Timestamp:     "2026-07-27T00:00:00Z",
	}
	document := buildCompatibilityVEX([]projection{{Statement: statement}}, time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))
	if len(document.Statements) != 1 {
		t.Fatalf("len(statements) = %d, want 1", len(document.Statements))
	}
	got := document.Statements[0]
	if got.Status != statement.Status || got.Justification != statement.Justification || got.Timestamp != statement.Timestamp {
		t.Fatalf("projected statement changed Docker's decision: %#v", got)
	}
}

func assertEnvironmentValue(t *testing.T, env []string, key, want string) {
	t.Helper()
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			if got := strings.TrimPrefix(entry, prefix); got != want {
				t.Fatalf("%s = %q, want %q", key, got, want)
			}
			return
		}
	}
	t.Fatalf("%s is missing from environment", key)
}

func assertEnvironmentMissing(t *testing.T, env []string, key string) {
	t.Helper()
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			t.Fatalf("%s unexpectedly present in environment", key)
		}
	}
}
