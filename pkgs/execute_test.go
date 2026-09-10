package pkgs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pc "github.com/tyklabs/packagecloud/api/v1"
)

var execNow = time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)

// execFixture returns one announced listing, its live counterpart and
// a store already holding the archived copy.
func execFixture(name, version string) (PlanPackage, pc.PackageDetail, *fakeStore) {
	content := []byte(name + "-" + version)
	filename := name + "_" + version + "_amd64.deb"
	pp := PlanPackage{
		Name:          name,
		Version:       version,
		Arch:          "amd64",
		DistroVersion: "ubuntu/jammy",
		Filename:      filename,
		Sha256Sum:     shaOf(content),
	}
	item := pc.PackageDetail{
		Name:          name,
		Version:       version,
		Arch:          "amd64",
		DistroVersion: "ubuntu/jammy",
		Filename:      filename,
		Sha256Sum:     shaOf(content),
	}
	store := newFakeStore()
	store.shas["tyk-test-repo/ubuntu/jammy/"+filename] = pp.Sha256Sum
	return pp, item, store
}

// recordingDeleter records deletions and optionally fails
type recordingDeleter struct {
	deleted []string
	fail    bool
}

func (d *recordingDeleter) delete(item pc.PackageDetail) error {
	if d.fail {
		return fmt.Errorf("api says no")
	}
	d.deleted = append(d.deleted, item.Filename)
	return nil
}

func execPlans(pp PlanPackage) (announced, fresh Plan) {
	announced = Plan{
		Repo:        "tyk-test-repo",
		GeneratedAt: execNow.Add(-31 * 24 * time.Hour),
		NotBefore:   execNow.Add(-24 * time.Hour),
		Packages:    []PlanPackage{pp},
	}
	fresh = Plan{Repo: "tyk-test-repo", Packages: []PlanPackage{pp}}
	return announced, fresh
}

func TestExecutePlanDeletes(t *testing.T) {
	pp, item, store := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	del := &recordingDeleter{}

	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, store, del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.True(t, res.Clean())
	assert.Equal(t, 1, res.Deleted)
	assert.Equal(t, 0, res.Skipped)
	assert.Equal(t, []string{pp.Filename}, del.deleted)
	require.Len(t, res.Packages, 1)
	assert.Equal(t, ActionDeleted, res.Packages[0].Action)
}

func TestExecutePlanDryRunDeletesNothing(t *testing.T) {
	pp, item, store := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	del := &recordingDeleter{}

	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, store, del.delete,
		ExecuteConfig{Delete: false, Now: execNow})
	require.NoError(t, err)
	assert.True(t, res.DryRun)
	assert.Equal(t, 1, res.WouldDelete)
	assert.Equal(t, 0, res.Deleted)
	assert.Empty(t, del.deleted)
	require.Len(t, res.Packages, 1)
	assert.Equal(t, ActionWouldDelete, res.Packages[0].Action)
}

func TestExecutePlanRefusesBeforeNotBefore(t *testing.T) {
	pp, item, store := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	announced.NotBefore = execNow.Add(24 * time.Hour) // grace not elapsed
	del := &recordingDeleter{}

	_, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, store, del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_before")
	assert.Empty(t, del.deleted)

	// the same plan is fine as a dry run
	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, store, del.delete,
		ExecuteConfig{Delete: false, Now: execNow})
	require.NoError(t, err)
	assert.Equal(t, 1, res.WouldDelete)
	assert.Empty(t, del.deleted)
}

func TestExecutePlanSkipsWhenFreshPlanDisagrees(t *testing.T) {
	pp, item, store := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	// an exception was granted during the grace window: the fresh
	// plan no longer prunes the listing
	fresh.Packages = nil
	del := &recordingDeleter{}

	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, store, del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Deleted)
	assert.Equal(t, 1, res.Skipped)
	assert.Empty(t, del.deleted)
	require.Len(t, res.Packages, 1)
	assert.Contains(t, res.Packages[0].Reason, "freshly derived plan")
}

func TestExecutePlanSkipsWhenGoneFromRegistry(t *testing.T) {
	pp, _, store := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	del := &recordingDeleter{}

	res, err := ExecutePlan(context.Background(), announced, fresh, nil, store, del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Skipped)
	assert.Contains(t, res.Packages[0].Reason, "not on packagecloud")
}

func TestExecutePlanSkipsChangedChecksum(t *testing.T) {
	pp, item, store := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	// same filename on the registry, different content: identity is
	// the sha, so the listing no longer matches the announcement
	item.Sha256Sum = shaOf([]byte("republished"))
	del := &recordingDeleter{}

	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, store, del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Deleted)
	assert.Equal(t, 1, res.Skipped)
	assert.Empty(t, del.deleted)
}

func TestExecutePlanSkipsUnarchived(t *testing.T) {
	pp, item, _ := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	del := &recordingDeleter{}

	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, newFakeStore(), del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Skipped)
	assert.Empty(t, del.deleted)
	assert.Contains(t, res.Packages[0].Reason, "not confirmed archived")
}

func TestExecutePlanSkipsArchiveMismatch(t *testing.T) {
	pp, item, store := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	store.shas["tyk-test-repo/ubuntu/jammy/"+pp.Filename] = shaOf([]byte("other"))
	del := &recordingDeleter{}

	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, store, del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Skipped)
	assert.Empty(t, del.deleted)
	assert.Contains(t, res.Packages[0].Reason, "does not match")
}

func TestExecutePlanFIPSDeletedWithoutArchive(t *testing.T) {
	pp, item, _ := execFixture("tyk-gateway-fips", "5.2.0")
	announced, fresh := execPlans(pp)
	del := &recordingDeleter{}

	// empty store: a non-FIPS package would be skipped here
	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, newFakeStore(), del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Deleted)
	assert.Equal(t, 1, res.Fips, "FIPS deletions must be flagged in the report")
	require.Len(t, res.Packages, 1)
	assert.True(t, res.Packages[0].Fips)
	assert.Equal(t, []string{pp.Filename}, del.deleted)
}

func TestExecutePlanRecordsFailedDeletes(t *testing.T) {
	pp, item, store := execFixture("tyk-test", "1.2.3")
	announced, fresh := execPlans(pp)
	del := &recordingDeleter{fail: true}

	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item}, store, del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.False(t, res.Clean())
	assert.Equal(t, 1, res.Failed)
	assert.Equal(t, ActionFailed, res.Packages[0].Action)
}

func TestExecutePlanPerDistroListings(t *testing.T) {
	// one file, two distro listings: both must be deleted separately
	pp, item, store := execFixture("tyk-test", "1.2.3")
	pp2 := pp
	pp2.DistroVersion = "el/9"
	item2 := item
	item2.DistroVersion = "el/9"
	store.shas["tyk-test-repo/el/9/"+pp2.Filename] = pp2.Sha256Sum

	announced, fresh := execPlans(pp)
	announced.Packages = append(announced.Packages, pp2)
	fresh.Packages = append(fresh.Packages, pp2)
	del := &recordingDeleter{}

	res, err := ExecutePlan(context.Background(), announced, fresh, []pc.PackageDetail{item, item2}, store, del.delete,
		ExecuteConfig{Delete: true, Now: execNow})
	require.NoError(t, err)
	assert.Equal(t, 2, res.Deleted)
	assert.Len(t, del.deleted, 2)
}

func TestExecuteResultRender(t *testing.T) {
	res := ExecuteResult{
		Repo: "tyk-test-repo", DryRun: true, WouldDelete: 3, Skipped: 1, Fips: 1,
		Packages: []ExecutedPackage{{Action: ActionSkipped, Reason: "not confirmed archived"}},
	}
	out := res.Render()
	assert.Contains(t, out, "would delete")
	assert.Contains(t, out, "not confirmed archived")
	assert.NotContains(t, out, "DELETED")
}
