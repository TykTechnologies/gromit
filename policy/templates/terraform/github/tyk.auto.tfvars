tyk_release_branches = [
{{- range $repo, $repoValue := .Repos }}
{{- if eq $repo "tyk" }}
	{{- range $branch, $values := $repoValue.Branches.Branch }}
  { branch    = "{{ $branch }}",
    reviewers = 2,
    convos    = false,
  	required_tests = ["Go 1.16 Redis 5"] },
	{{- end }}
{{- end }}
{{- end }}
] 