portal_release_branches = [
{{- range $repo, $repoValue := .Repos }}
{{- if eq $repo "portal" }}
	{{- range $branch, $values := $repoValue.Branches.Branch }}
  { branch    = "{{ $branch }}",
    reviewers = 1,
    convos    = false,
    required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)"] },
	{{- end }}
{{- end }}
{{- end }}
]