tyk_release_branches = [
{{- with $repo := index .RepoPolicies "tyk" }}
{{- range $branch, $values := $repo.ReleaseBranches }}
{ branch    = "{{ $branch }}",
	reviewers = "{{ $values.ReviewCount }}",
	convos    = "{{ $values.Convos }}",
	{{- if $values.SourceBranch }}
	source_branch  = "{{ $values.SourceBranch }}",
	{{- end }}
	required_tests = ["{{ $values.Tests | join "," }}"] },
{{- end }}
{{- end }}
]