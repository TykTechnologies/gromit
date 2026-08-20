# Trivy: stale OCI VEX can override a live repository decision

## Summary

Trivy 0.72 combines repeated VEX sources with suppressive logical-OR
semantics. When scanning with:

```bash
trivy image --vex repo --vex oci --show-suppressed IMAGE
```

a stale OCI `not_affected` or `fixed` statement suppresses a finding even when
the live VEX repository, queried first, contains a newer `affected` or
`under_investigation` decision for the same vulnerability and package.

Changing the CLI source order does not make a non-suppressing decision veto a
later source. Trivy does not compare timestamps, versions, authors, signatures,
or source authority across VEX sources.

This makes it unsafe to combine a continuously updated repository with a
digest-bound OCI VEX snapshot as an offline fallback.

## Affected version

- Trivy: `0.72.0`
- Source commit:
  `8a32853686209a428179bb3a1688802b25691564`
- `go-vex`: `v0.2.7`, commit
  `3185a64ed27703fc3fe4af8cd5e1ce0ed2fa2569`

As of 2026-07-29, `v0.72.0` is still the latest non-prerelease Trivy
release. Trivy `main` at
`990d76568ecab5583381facd112bfd5ac6f4266b` retains the same
`Client.NotAffected` loop: it returns on the first source that grants
suppression, while a non-suppressing result does not veto later sources.

## Minimal production-client reproduction

At the exact Trivy commit above, add
`pkg/vex/source_order_regression_test.go`:

```go
package vex_test

import (
	"testing"

	"github.com/aquasecurity/trivy/pkg/sbom/core"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/aquasecurity/trivy/pkg/vex"
)

type fakeSource struct {
	label       string
	notAffected bool
}

func (s fakeSource) NotAffected(
	_ types.DetectedVulnerability,
	_, _ *core.Component,
) (types.ModifiedFinding, bool) {
	return types.ModifiedFinding{Source: s.label}, s.notAffected
}

func TestClientNotAffectedCrossSourceOrder(t *testing.T) {
	repository := fakeSource{label: "repo", notAffected: false}
	oci := fakeSource{label: "oci", notAffected: true}

	tests := []struct {
		name    string
		sources []vex.VEX
	}{
		{name: "repo_then_oci", sources: []vex.VEX{repository, oci}},
		{name: "oci_then_repo", sources: []vex.VEX{oci, repository}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := vex.Client{VEXes: test.sources}
			modified, suppressed := client.NotAffected(
				types.DetectedVulnerability{},
				nil,
				nil,
			)
			if !suppressed {
				t.Fatal("Client.NotAffected() returned false, want true")
			}
			if got, want := modified.Source, "oci"; got != want {
				t.Fatalf("source = %q, want %q", got, want)
			}
		})
	}
}
```

Run:

```bash
go test ./pkg/vex \
  -run TestClientNotAffectedCrossSourceOrder \
  -count=1 -v
```

Observed:

```text
=== RUN   TestClientNotAffectedCrossSourceOrder
=== RUN   TestClientNotAffectedCrossSourceOrder/repo_then_oci
=== RUN   TestClientNotAffectedCrossSourceOrder/oci_then_repo
--- PASS: TestClientNotAffectedCrossSourceOrder (0.00s)
    --- PASS: TestClientNotAffectedCrossSourceOrder/repo_then_oci (0.00s)
    --- PASS: TestClientNotAffectedCrossSourceOrder/oci_then_repo (0.00s)
PASS
ok  	github.com/aquasecurity/trivy/pkg/vex	0.692s
```

This test invokes the production `Client.NotAffected` implementation through
the production `vex.VEX` interface. It proves that a repository-like
non-suppressing result followed by an OCI-like suppressing result is
suppressed and attributed to OCI. Reversing source order still suppresses and
attributes OCI.

The test does not instantiate repository/OCI loaders or perform an image scan.
An end-to-end fixture would additionally prove transport loading, but it
cannot change the production client result demonstrated here.

## Expected behavior

A current authoritative non-suppressing decision must be able to veto an older
or lower-authority suppressing decision. At minimum, Trivy needs an explicit
conflict policy for multiple VEX sources rather than treating all
`not_affected` and `fixed` statements as independent suppression grants.

Possible contracts include:

1. strict source precedence, where any matching statement in a higher-priority
   source is final, including `affected` and `under_investigation`;
2. a conservative conflict mode where any matching non-suppressing decision
   keeps the finding active;
3. an explicit policy flag that selects source precedence or conservative
   conflict handling without silently changing existing behavior.

Cross-source timestamp comparison alone is insufficient unless Trivy also
defines trusted authorship and clock semantics.

## Source analysis

At the affected commit:

- `pkg/flag/vulnerability_flags.go` preserves repeated `--vex` values;
- `pkg/vex/vex.go` initializes sources in CLI order;
- `pkg/vex.Client.NotAffected` returns on the first source that returns true,
  but false does not veto later sources;
- `pkg/vex/repo.go` applies precedence only between repositories inside the
  repository source;
- `pkg/vex/openvex.go` treats `not_affected` and `fixed` as suppressing, while
  `affected` and `under_investigation` return false;
- `pkg/vex/oci.go` consumes the first discovered OCI VEX document.

Trivy's `go test ./pkg/vex` passes at the inspected commit.

## Existing Issue Search

On 2026-07-29, GitHub issue searches in `aquasecurity/trivy` returned no
matches for:

- `VEX OCI repository conflict`;
- `VEX precedence`;
- `stale VEX`; and
- `under_investigation not_affected VEX`.

This search is not proof that no semantically related report exists, but no
direct duplicate was found. This document remains a prepared report; no Trivy
issue has been posted from this workspace.

## Security impact

An immutable image can outlive its original VEX decision. If the publisher
later changes a decision from `not_affected` to `affected`, a customer that
mirrored the image and its older OCI attestation continues to suppress the
finding even while using the publisher's current live VEX repository.

The scanner output appears auditable because `--show-suppressed` identifies
the OCI source, but it does not warn that a live conflicting decision was
ignored.
