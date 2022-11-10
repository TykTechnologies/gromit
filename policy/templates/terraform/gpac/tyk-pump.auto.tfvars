tyk-pump_release_branches = [
<<<<<<< HEAD:policy/terraform/github/tyk-pump.auto.tfvars
  { branch    = "master",
    reviewers = 2,
    convos    = false,
  required_tests = [] },
]
=======
{{- with $repo := index .RepoPolicies "tyk-pump" }}
{{- range $branch, $values := $repo.ReleaseBranches }}
{ branch    = "{{ $branch }}",
	reviewers = "{{ $values.ReviewCount }}",
	convos    = "{{ $values.Convos }}",
	{{- if $values.SourceBranch }}
	source_branch  = "{{ $values.SourceBranch }}",
	{{- end }}
	required_tests = [{{ range $index, $test := $values.Tests }}{{ if $index }},{{ end }}"{{ $test }}"{{ end }}]},
{{- end }}
{{- end }}
]
>>>>>>> ec909c4 (Cleanup):policy/templates/terraform/gpac/tyk-pump.auto.tfvars
