tyk_release_branches = [
{{- range $repo, $repoValue := .Repos }}
{{- if eq $repo "tyk" }}
	{{- $branches := $repoValue.Branches -}}
	{{- range $branch, $values := $repoValue.Branches.Branch }}
  { branch    = "{{ $branch }}",
    reviewers = "{{ or $values.ReviewCount $branches.ReviewCount }}",
    convos    = "{{ or $values.Convos $branches.Convos }}",
    {{- if $values.SourceBranch }}
    source_branch  = "{{ $values.SourceBranch }}",
    {{- end }}
  	required_tests = [{{ or $branches.Tests $values.Tests | join "," }}] },
	{{- end }}
{{- end }}
{{- end }}
]