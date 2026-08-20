package policy

import (
	"reflect"
	"strings"

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
		if len(a.Docker) > 0 && !a.SkipDocker {
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

func (pc pluginCompilerConfig) GetGoImage(buildenv string) string {
	if pc.GoImage != "" {
		return pc.GoImage
	}
	return buildenv
}

func (pc pluginCompilerConfig) PullRequestTypesInline() bool {
	return pc.PullRequestTypesStyle == "inline"
}

func (pc pluginCompilerConfig) DockerBuildIfMultiline() bool {
	return strings.Contains(pc.DockerBuildIf, "\n")
}

func (v pluginCompilerVariant) MetadataID() string {
	if v.Name == "" || v.Name == "std" {
		return "set-metadata"
	}
	return "set-metadata-" + v.Name
}

func (v pluginCompilerVariant) PublishID() string {
	if v.Name == "" || v.Name == "std" {
		return "build-push-ng"
	}
	return "build-push-" + v.Name + "-ng"
}

func (v pluginCompilerVariant) StepSuffix() string {
	if v.Name == "" || v.Name == "std" {
		return ""
	}
	return " " + strings.ToUpper(v.Name)
}

func (v pluginCompilerVariant) ImageName() string {
	return "tyk-plugin-compiler" + v.ImageSuffix
}

func (v pluginCompilerVariant) NextGenEdition() string {
	if v.Edition != "" {
		return v.Edition
	}
	switch v.Name {
	case "ee":
		return "ee"
	case "fips":
		return "ee-fips"
	default:
		return "ce"
	}
}

func (v pluginCompilerVariant) NextGenTitle(tagSuffix string) string {
	if v.Title != "" {
		return v.Title + tagSuffix
	}
	return v.ImageName() + tagSuffix
}

func (v pluginCompilerVariant) BuildArgsBlock() string {
	var args []string
	if v.BuildTag != "" {
		args = append(args, "BUILD_TAG="+v.BuildTag)
	}
	if v.GoExperiment != "" {
		args = append(args, "GOEXPERIMENT="+v.GoExperiment)
	}
	if v.GoFips140 != "" {
		args = append(args, "GOFIPS140="+v.GoFips140)
	}
	if len(args) == 0 {
		return ""
	}
	return "\n            " + strings.Join(args, "\n            ")
}

func (pc pluginCompilerConfig) NextGenVariants() []pluginCompilerVariant {
	if len(pc.NextGen.Variants) > 0 {
		return pc.NextGen.Variants
	}
	return pc.Variants
}

func (ng pluginCompilerNextGenConfig) TagSuffixValue() string {
	if ng.TagSuffix != "" {
		return ng.TagSuffix
	}
	return "-ng"
}

func (ng pluginCompilerNextGenConfig) BaseImageNameValue() string {
	if ng.BaseImageName != "" {
		return ng.BaseImageName
	}
	return "tyk-plugin-compiler-ng-base"
}

func (ng pluginCompilerNextGenConfig) BaseImageTagValue() string {
	if ng.BaseImageTag != "" {
		return ng.BaseImageTag
	}
	return "latest"
}

func (ng pluginCompilerNextGenConfig) BaseImageRefValue() string {
	if ng.BaseImageRef != "" {
		return ng.BaseImageRef
	}
	return "tykio/" + ng.BaseImageNameValue() + ":" + ng.BaseImageTagValue()
}

func (ng pluginCompilerNextGenConfig) WorkflowBaseImageNameValue() string {
	if ng.WorkflowBaseImageName != "" {
		return ng.WorkflowBaseImageName
	}
	return "tyk-plugin-compiler"
}

func (ng pluginCompilerNextGenConfig) WorkflowBaseImageTagValue() string {
	if ng.WorkflowBaseImageTag != "" {
		return ng.WorkflowBaseImageTag
	}
	return "ng-base-${{ github.sha }}"
}

func (ng pluginCompilerNextGenConfig) BaseImageValue() string {
	if ng.BaseImage != "" {
		return ng.BaseImage
	}
	return "dhi.io/busybox:1-debian-fips-dev"
}

func (ng pluginCompilerNextGenConfig) BaseSourceImageRefValue() string {
	ref := ng.BaseImageValue()
	if ng.BaseImageDigest != "" {
		return ref + "@" + ng.BaseImageDigest
	}
	return ref
}

func (ng pluginCompilerNextGenConfig) GlibcTargetValue() string {
	if ng.GlibcTarget != "" {
		return ng.GlibcTarget
	}
	return "2.17"
}

func (ng pluginCompilerNextGenConfig) SlimValue() string {
	if ng.Slim != "" {
		return ng.Slim
	}
	return "0"
}

func (ng pluginCompilerNextGenConfig) WithCxxValue() string {
	if ng.WithCxx != "" {
		return ng.WithCxx
	}
	return "1"
}

func (ng pluginCompilerNextGenConfig) WithGitValue() string {
	if ng.WithGit != "" {
		return ng.WithGit
	}
	return "1"
}

func (ng pluginCompilerNextGenConfig) CrossValue() string {
	if ng.Cross != "" {
		return ng.Cross
	}
	return "1"
}

func (ng pluginCompilerNextGenConfig) GatewayTrimpathValue() string {
	if ng.GatewayTrimpath != "" {
		return ng.GatewayTrimpath
	}
	return "true"
}

func (ng pluginCompilerNextGenConfig) CEArchsValue() string {
	if ng.CEArchs != "" {
		return ng.CEArchs
	}
	return "amd64,arm64,s390x"
}

func (ng pluginCompilerNextGenConfig) EEArchsValue() string {
	if ng.EEArchs != "" {
		return ng.EEArchs
	}
	return ng.CEArchsValue()
}

func (ng pluginCompilerNextGenConfig) FIPSArchsValue() string {
	if ng.FIPSArchs != "" {
		return ng.FIPSArchs
	}
	return ng.EEArchsValue()
}

func (ng pluginCompilerNextGenConfig) VariantBuildArgsBlock(v pluginCompilerVariant) string {
	var args []string
	edition := v.NextGenEdition()
	args = append(args, "DEFAULT_EDITION="+edition)
	switch edition {
	case "ee":
		args = append(args, "EE_AVAILABLE=true")
		if v.BuildTag != "" {
			args = append(args, "EE_BUILD_TAG="+v.BuildTag)
		}
	case "ee-fips":
		args = append(args, "FIPS_AVAILABLE=true")
		if v.BuildTag != "" {
			args = append(args, "FIPS_BUILD_TAG="+v.BuildTag)
		}
		if v.GoFips140 != "" {
			args = append(args, "FIPS_GOFIPS140="+v.GoFips140)
		}
		if v.GoExperiment != "" {
			args = append(args, "FIPS_GOEXPERIMENT="+v.GoExperiment)
		}
	}
	if v.BuildTag != "" {
		args = append(args, "SELFTEST_BUILD_TAG="+v.BuildTag)
	}
	if v.GoExperiment != "" {
		args = append(args, "SELFTEST_GOEXPERIMENT="+v.GoExperiment)
	}
	if v.GoFips140 != "" {
		args = append(args, "SELFTEST_GOFIPS140="+v.GoFips140)
	}
	if len(args) == 0 {
		return ""
	}
	return "\n            " + strings.Join(args, "\n            ")
}

// CustomizationOrgValue is the Docker Hub organization owning the DHI
// customization. It defaults to the organization prefix of BaseImage so the
// common case needs no extra configuration.
func (ng pluginCompilerNextGenConfig) CustomizationOrgValue() string {
	if ng.CustomizationOrg != "" {
		return ng.CustomizationOrg
	}
	// BaseImageValue applies the configured default; reading the raw field would
	// yield "" when only a default is set.
	if org, _, found := strings.Cut(ng.BaseImageValue(), "/"); found {
		// A registry host is not a Docker Hub organization. Returning one would
		// render `--org dhi.io`, so leave it empty and let the caller's guard fire.
		if !strings.Contains(org, ".") && !strings.Contains(org, ":") {
			return org
		}
	}
	return ""
}

// ImagePlatformsValue is the platform list the compiler image is published for.
// This is the image's own architecture, not the architectures a plugin can be
// built for -- see CEArchsValue and friends for those.
func (ng pluginCompilerNextGenConfig) ImagePlatformsValue() string {
	if ng.ImagePlatforms != "" {
		return ng.ImagePlatforms
	}
	return "linux/amd64"
}

// GatePlatformValue is the platform that runs the full compile-and-load gate.
// It must appear in ImagePlatformsValue; a gate build uses `load: true`, which
// accepts a single platform only.
func (ng pluginCompilerNextGenConfig) GatePlatformValue() string {
	if ng.GatePlatform != "" {
		return ng.GatePlatform
	}
	return "linux/amd64"
}

// ImageNeedsGoTarball reports whether any requested image platform has to fetch
// the Go toolchain from go.dev instead of copying it out of golang-cross, which
// is published for amd64 only. When it is false the workflow makes no go.dev
// call at all, so an amd64-only build gains no new network dependency.
func (ng pluginCompilerNextGenConfig) ImageNeedsGoTarball() bool {
	for _, p := range strings.Split(ng.ImagePlatformsValue(), ",") {
		if p = strings.TrimSpace(p); p != "" && p != "linux/amd64" {
			return true
		}
	}
	return false
}

// GatesExtraPlatforms reports whether this variant gets the Tier B toolchain
// gate. With ExtraGateVariant set, exactly one variant does; the others publish
// their extra platforms on the strength of that one plus the full gate they
// each get on GatePlatformValue.
func (ng pluginCompilerNextGenConfig) GatesExtraPlatforms(v pluginCompilerVariant) bool {
	if ng.ExtraGateVariant == "" {
		return true
	}
	name := v.Name
	if name == "" {
		name = "std"
	}
	return name == ng.ExtraGateVariant
}

// ExtraImagePlatforms is ImagePlatformsValue minus GatePlatformValue: the
// platforms that are published but do not get the full gate.
func (ng pluginCompilerNextGenConfig) ExtraImagePlatforms() []string {
	gate := ng.GatePlatformValue()
	var extra []string
	for _, p := range strings.Split(ng.ImagePlatformsValue(), ",") {
		if p = strings.TrimSpace(p); p != "" && p != gate {
			extra = append(extra, p)
		}
	}
	return extra
}
