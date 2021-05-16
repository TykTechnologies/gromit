package policy

import (
	"bytes"
	"fmt"
)

type branchPolicies struct {
	Protected    []string            `mapstructure:",omitempty"`
	Deprecations map[string][]string `mapstructure:",omitempty"`
	Backports    map[string]string   `mapstructure:",omitempty"`
	Files        []string            `mapstructure:",omitempty"`
}

type RepoPolicies struct {
	Protected []string
	Repos     map[string]branchPolicies
	Files     []string
}

// String representation
func (r RepoPolicies) String() string {
	w := new(bytes.Buffer)
	fmt.Fprintln(w, `Commits landing on the Source branch are automatically sync'd to the list of Destinations. PRs will be created for the protected branch. Other branches will be updated directly.`)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Protected branches: %v\n", r.Protected)
	fmt.Fprintln(w, "Common Files:")
	for _, file := range r.Files {
		fmt.Fprintf(w, " - %s\n", file)
	}
	for repo, pols := range r.Repos {
		fmt.Fprintf(w, "%s\n", repo)
		fmt.Fprintln(w, " Extra files:")
		for _, f := range pols.Files {
			fmt.Fprintf(w, "   - %s\n", f)
		}
		fmt.Fprintln(w, " Deprecations:")
		for version, files := range pols.Deprecations {
			fmt.Fprintf(w, "  Version %s\n", version)
			for _, f := range files {
				fmt.Fprintf(w, "   - %s\n", f)
			}
		}
		fmt.Fprintln(w, " Backports")
		for src, dest := range pols.Backports {
			fmt.Fprintf(w, "   - %s → %s\n", src, dest)
		}
	}
	fmt.Fprintln(w)
	return w.String()
}
