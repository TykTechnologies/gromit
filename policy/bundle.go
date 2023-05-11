package policy

import (
	"embed"
	"fmt"
	"io"
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

//go:embed template-features all:template-features
var Features embed.FS

// bundleNode of a directory tree
type bundleNode struct {
	Name     string
	path     string
	Children []*bundleNode
}

// Bundle represents a directory tree, it can be instantiated by NewBundle()
type Bundle struct {
	Name string
	bfs  fs.FS
	tree *bundleNode
}

// Add adds the path into the templateNode tree
// This code due to ChatGPT
func (b *Bundle) Add(path string) {
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
func (b *Bundle) Render(bv any, opDir string, n *bundleNode) ([]string, error) {
	var renderedFiles []string
	if n == nil {
		n = b.tree
	}
	if strings.HasSuffix(n.Name, ".d") {
		return nil, nil
	}
	for _, child := range n.Children {
		f, err := b.Render(bv, opDir, child)
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, f...)
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
		var op io.Writer
		opFile := filepath.Join(opDir, n.path)
		if strings.HasPrefix(opFile, "-") {
			op = io.Writer(os.Stdout)
		} else {
			dir, _ := filepath.Split(opFile)
			err := os.MkdirAll(dir, 0755)
			if err != nil && !os.IsExist(err) {
				return nil, err
			}
			opf, err := os.Create(opFile)
			if err != nil {
				return nil, err
			}
			defer opf.Close()
			op = io.Writer(opf)
		}
		err := t.Execute(op, bv)
		if err != nil {
			return nil, fmt.Errorf("rendering to %s: %v", opFile, err)
		}
		renderedFiles = append(renderedFiles, n.path)
	}
	return renderedFiles, nil
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
func (b *Bundle) String() string {
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
// Also traverses template-features/<feature> and adds it to the bundle
func NewBundle(bundleName string, features []string) (*Bundle, error) {
	var bfs fs.FS
	if strings.HasPrefix(bundleName, ".") || strings.HasPrefix(bundleName, "/") {
		bfs = os.DirFS(bundleName)
	} else {
		var err error
		bfs, err = fs.Sub(Bundles, filepath.Join("templates", bundleName))
		if err != nil {
			log.Fatal().Err(err).Msg("fetching embedded templates")
		}
	}
	b := &Bundle{Name: bundleName,
		bfs:  bfs,
		tree: &bundleNode{}}

	err := fs.WalkDir(bfs, ".", func(path string, d fs.DirEntry, err error) error {
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
	if err != nil {
		return b, err
	}
	// Walk the features
	for _, feat := range features {
		var ffs fs.FS
		if strings.HasPrefix(feat, ".") || strings.HasPrefix(feat, "/") {
			ffs = os.DirFS(feat)
		} else {
			var err error
			ffs, err = fs.Sub(Features, filepath.Join("template-features", bundleName, feat))
			if err != nil {
				log.Fatal().Err(err).Msgf("fetching embedded feature %s", feat)
			}
		}
		err := fs.WalkDir(ffs, ".", func(path string, d fs.DirEntry, err error) error {
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
		if err != nil {
			return b, err
		}		
	}
	
	return b, err
}
