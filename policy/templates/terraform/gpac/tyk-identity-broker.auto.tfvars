tyk-identity-broker_release_branches = [
<<<<<<< HEAD:policy/terraform/github/tyk-identity-broker.auto.tfvars
  { branch    = "master",
    reviewers = 1,
    convos    = false,
  required_tests = [] },
]
=======
{{- with $repo := index .RepoPolicies "tyk-identity-broker" }}
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
>>>>>>> ec909c4 (Cleanup):policy/templates/terraform/gpac/tyk-identity-broker.auto.tfvars
