tyk-pump_release_branches = [
{{- range $repo, $repoValue := .Repos }}
{{- if eq $repo "tyk-pump" }}
	{{- range $branch, $values := $repoValue.Branches.Branch }}
  { branch    = "{{ $branch }}",
    reviewers = 2,
    convos    = false,
  required_tests = [] },
	{{- end }}
{{- end }}
{{- end }}
]