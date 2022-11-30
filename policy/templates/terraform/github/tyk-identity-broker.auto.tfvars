tyk-identity-broker_release_branches = [
{{- range $branch, $values := .ReleaseBranches }}
{ branch    = "{{ $branch }}",
	reviewers = "{{ $values.ReviewCount }}",
	convos    = "{{ $values.Convos }}",
	{{- if $values.SourceBranch }}
	source_branch  = "{{ $values.SourceBranch }}",
	{{- end }}
	required_tests = ["{{ $values.Tests | join "," }}"] },
{{- end }}
]