package policy

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/rs/zerolog/log"
)

//go:embed templates all:templates
var Bundles embed.FS

// FIXME: the below is for compatibilty, remove when appropriate
//
//go:embed templates all:templates
var templates embed.FS

// bundleNode of a directory tree
type bundleNode struct {
	Name     string
	path     string
	Children []*bundleNode
}

// bundle is a private type representing a directory tree, it can be instantiated by NewBundle()
type bundle struct {
	Name string
	bfs  fs.FS
	tree *bundleNode
}

// Add adds the path into the templateNode tree
// This code due to ChatGPT
func (b *bundle) Add(path string) {
	// Split the path into its components
	components := strings.Split(path, string(os.PathSeparator))

	// Find the parent node
	parent := b.tree
	for i := 0; i < len(components)-1; i++ {
		found := false
		c := components[i]
		for _, child := range parent.Children {
			if child.Name == c {
				parent = child
				found = true
				break
			}
		}
		if !found {
			newNode := &bundleNode{Name: c, path: path}
			parent.Children = append(parent.Children, newNode)
			parent = newNode
		}
	}
	parent.Children = append(parent.Children, &bundleNode{Name: components[len(components)-1], path: path})
}

// Render will walk a tree given in n, depth first, skipping .d nodes
// All leaf nodes will be rendered
func (b *bundle) Render(bv BundleVars, opDir string, n *bundleNode) error {
	if n == nil {
		n = b.tree
	}
	if strings.HasSuffix(n.Name, ".d") {
		return nil
	}
	for _, child := range n.Children {
		err := b.Render(bv, opDir, child)
		if err != nil {
			return err
		}
	}
	if len(n.Children) == 0 {
		templatePaths := b.tree.findSubTemplates(n.Name + ".d")
		// The top-level template path is required in the list of paths
		templatePaths = append(templatePaths, n.path)
		log.Debug().Strs("subtemplates", templatePaths).Str("template", n.Name).Msg("rendering")
		t := template.Must(
			template.New(n.Name).
				Funcs(sprig.TxtFuncMap()).
				Option("missingkey=error").
				ParseFS(b.bfs, templatePaths...))
		return bv.renderTemplate(t, filepath.Join(opDir, n.path))
	}
	return nil
}

// findSubTemplates finds subtemplates anywhere in the parsed tree but
// it should be called on leaf nodes only
func (n *bundleNode) findSubTemplates(name string) []string {
	if name == n.Name {
		var subTemplates []string
		for _, child := range n.Children {
			subTemplates = append(subTemplates, child.path)
		}
		return subTemplates
	}
	for _, child := range n.Children {
		st := child.findSubTemplates(name)
		if len(st) > 0 {
			return st
		}
	}

	return nil
}

// String will provide a human readable bundle listing
func (b *bundle) String() string {
	return fmt.Sprintf(b.Name) + b.tree.String(0)
}

// Print an indented tree
func (n *bundleNode) String(indent int) string {
	op := fmt.Sprintf("%s%s\n", strings.Repeat("  ", indent), n.Name)
	for _, child := range n.Children {
		op += child.String(indent + 1)
	}
	return op
}

// Returns a bundle by traversing from templates/<bundleDir>
func NewBundle(bfs fs.FS, bundleName string) (*bundle, error) {
	b := &bundle{Name: bundleName,
		bfs:  bfs,
		tree: &bundleNode{}}

	return b, fs.WalkDir(bfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root node
		if path == "." {
			return nil
		}
		// Normalize the path to use '/' as the separator
		path = strings.ReplaceAll(path, string(os.PathSeparator), "/")
		b.Add(path)
		return nil
	})
}
