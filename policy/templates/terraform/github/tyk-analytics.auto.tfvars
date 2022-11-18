tyk-analytics_release_branches = [
{{- range $repo, $repoValue := .Repos }}
{{- if eq $repo "tyk-analytics" }}
	{{- range $branch, $values := $repoValue.Branches.Branch }}
  { branch    = "{{ $branch }}",
    reviewers = 0,
    convos    = false,
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
	{{- end }}
{{- end }}
{{- end }}
]