tyk-analytics-ui_release_branches = [
<<<<<<< HEAD:policy/terraform/github/tyk-analytics-ui.auto.tfvars
  { branch    = "master",
    reviewers = 2,
    convos    = false,
  required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
  { branch    = "release-4",
    reviewers = 0,
    convos    = false,
  required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
  { branch        = "release-4.3",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4",
  required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
  { branch        = "release-4.3.0",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4",
  required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
  { branch        = "release-4.3.1",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4.3",
  required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
  { branch        = "release-4.0.10",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4-lts",
  required_tests = [] },

]
=======
{{- with $repo := index .RepoPolicies "tyk-analytics-ui" }}
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
>>>>>>> ec909c4 (Cleanup):policy/templates/terraform/gpac/tyk-analytics-ui.auto.tfvars
