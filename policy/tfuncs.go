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
