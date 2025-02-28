package pkgs

import (
	"fmt"
	"sync"
	"time"

	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
	pc "github.com/tyklabs/packagecloud/api/v1"
	"golang.org/x/mod/semver"
)

type Filter struct {
	Exceptions  util.Set[string]
	Version     string
	Age         time.Duration
	DownloadAge time.Duration
	total       int
	filtered    int
	mu          sync.Mutex
}

//nolint:copylocks
func (f Filter) String() string {
	var exceptions string
	for e := range f.Exceptions {
		exceptions += e + " "
	}
	var str string
	if exceptions != "" {
		str = fmt.Sprintf("exceptions: [ %s]\n", exceptions)
	}
	if f.Version != "" {
		str += fmt.Sprintf("versions before %s ", semver.Canonical(f.Version))
	}
	if f.Age > 0 {
		str += fmt.Sprintf("uploaded before %s ", time.Now().Add(-f.Age).Format("Jan 2 2006"))
	}
	str += fmt.Sprintf("%d/%d filtered", f.filtered, f.total)
	return str
}

func (r Repos) MakeFilter(repoName string) (*Filter, error) {
	var f Filter
	repo, found := r[repoName]
	if !found {
		return &f, fmt.Errorf("%s not known among %v", repoName, r)
	}
	if repo.VersionCutoff != "" && !semver.IsValid(repo.VersionCutoff) {
		return &f, fmt.Errorf("%s cannot be parsed as semver", repo.VersionCutoff)
	}
	f.Version = repo.VersionCutoff
	f.Age = repo.AgeCutoff
	f.Exceptions = util.NewSetFromSlices(repo.Exceptions)

	return &f, nil
}

// Satisfies behaviour depends on the order of the conditionals:
// 1. Is it a version that should not be deleted?
// 2. Is it older than the versioncutoff?
// 3. Was the package uploaded before the agecutoff?
func (f *Filter) Satisfies(item pc.PackageDetail, now time.Time) bool {
	if f.Exceptions.Has(item.Version) {
		log.Trace().Msgf("v%s is protected", item.Version)
		return false
	}
	if f.Version != "" && semver.IsValid(item.Version) {
		if semver.Compare(f.Version, item.Version) <= 0 {
			return true
		} else {
			log.Trace().Msgf("%s v%s is newer than %s", item.Name, item.Version, f.Version)
		}
	}
	if f.Age != 0 {
		pAge := now.Sub(item.CreateTime)
		if pAge > f.Age {
			return true
		} else {
			log.Trace().Msgf("%s v%s created on %s younger than %s", item.Version, item.Version, item.CreateTime, f.Age)
		}
	}
	log.Trace().Interface("pkg", item).Msg("filtered out because no filter condition applies")
	return false
}

func (f *Filter) IncFiltered() {
	f.mu.Lock()
	f.filtered++
	f.mu.Unlock()
}

func (f *Filter) IncTotal() {
	f.mu.Lock()
	f.total++
	f.mu.Unlock()
}
