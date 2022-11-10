portal_release_branches = [
<<<<<<< HEAD:policy/terraform/github/portal.auto.tfvars
  { branch    = "master",
    reviewers = 1,
    convos    = false,
  required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)"] },
]
=======
{{- with $repo := index .RepoPolicies "portal" }}
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
>>>>>>> ec909c4 (Cleanup):policy/templates/terraform/gpac/portal.auto.tfvars
