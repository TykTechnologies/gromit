tyk-sink_release_branches = [
{{- range $repo, $repoValue := .Repos }}
{{- if eq $repo "tyk-sink" }}
	{{- range $branch, $values := $repoValue.Branches.Branch }}
  { branch    = "{{ $branch }}",
    reviewers = 1,
    convos    = false,
  required_tests = [] },
	{{- end }}
{{- end }}
{{- end }}
]