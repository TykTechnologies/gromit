package policy

import (
	"reflect"

	"github.com/TykTechnologies/gromit/util"
)

// Template functions called while rendering

// getCC returns the appropriate C compiler for the target architecture given the host architecture
func (rp RepoPolicy) GetCC(target, host string) string {
	if target != host {
		return target + "-linux-gnu-gcc"
	}
	return "gcc"
}

// getImages returns the list of container manifests
func (b *build) GetImages(repos ...string) []string {
	images := make(util.Set[string])
	for _, repo := range repos {
		image := getBuildField(b, repo)
		if len(image) > 0 {
			images.Add(image)
		}
	}
	return images.Members()
}

// getDockerPlatforms returns the list of docker platforms that are to be supported
func (b *build) GetDockerPlatforms() []string {
	platforms := make(util.Set[string])
	for _, a := range b.Archs {
		if len(a.Docker) > 0 {
			platforms.Add(a.Docker)
		}
	}
	return platforms.Members()
}

// getDockerBuilds returns a map of builds that have at least one container build
func (rp RepoPolicy) GetDockerBuilds() buildMap {
	dBuilds := make(buildMap)
	for b, bv := range rp.Branchvals.Builds {
		if bv.CIRepo != "" || bv.DHRepo != "" || bv.CSRepo != "" {
			dBuilds[b] = bv
		}
	}
	return dBuilds
}

// (rp RepoPolicy) HasBuild(build string) checks if the supplied build is defined
func (rp RepoPolicy) HasBuild(build string) bool {
	_, found := rp.Branchvals.Builds[build]
	return found
}

// getBuildField helps with accessing properties of the build type
func getBuildField(v *build, field string) string {
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(field)
	return f.String()
}
