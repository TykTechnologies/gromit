tyk-analytics-ui_release_branches = [
{{- range $repo, $repoValue := .Repos }}
{{- if eq $repo "tyk-analytics-ui" }}
	{{- range $branch, $values := $repoValue.Branches.Branch }}
  { branch    = "{{ $branch }}",
    reviewers = 0,
    convos    = false,
    required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
	{{- end }}
{{- end }}
{{- end }}
]