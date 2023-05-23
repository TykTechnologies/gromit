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

// Files common to all features
//go:embed templates all:templates
var Bundles embed.FS

// Extra files for a feature using the same tree structure as Bundles
// with a prefix <bundle>/<feature>/
//go:embed template-features all:template-features
var Features embed.FS

// bundleNode of a directory tree
type bundleNode struct {
	Name     string
	path     string
	template *template.Template
	Children []*bundleNode
}

// Bundle represents a directory tree, instantiated by NewBundle()
type Bundle struct {
	Name string
	tree *bundleNode
}

// Add adds the path and corresponding template into the templateNode tree
// This code due to ChatGPT
func (b *Bundle) Add(path string, template *template.Template) {
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
			newNode := &bundleNode{Name: c, path: path, template: template}
			parent.Children = append(parent.Children, newNode)
			parent = newNode
		}
	}
	parent.Children = append(parent.Children, &bundleNode{Name: components[len(components)-1], path: path, template: template})
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
		log.Debug().Str("template", n.Name).Msg("rendering")
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
		err := n.template.Execute(op, bv)
		if err != nil {
			return nil, fmt.Errorf("rendering to %s: %v", opFile, err)
		}
		renderedFiles = append(renderedFiles, n.path)
	}
	return renderedFiles, nil
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

// fsTreeWalk will walk the complete tree of tfs and add templates to the supplied Bundle b.
// Used to walk both the common bundle and the features bundle
func fsTreeWalk(b *Bundle, tfs fs.FS) error {
	err := fs.WalkDir(tfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root node
		if path == "." {
			return nil
		}
		// Do not recurse into sub-templates
		if (d.IsDir() && strings.HasSuffix(path, ".d")) {
			return fs.SkipDir
		}
		if ! d.IsDir() {
			templates := []string{path}

			stPath := path + ".d"
			fi, err := fs.Stat(tfs, stPath)
			if err == nil && fi.IsDir() {
				des, err := fs.ReadDir(tfs, stPath)
				if err != nil {
					return err
				}
				for _, de := range des {
					templates = append(templates, filepath.Join(stPath, de.Name()))
				}
			}
			// Normalize the path to use '/' as the separator
			path = strings.ReplaceAll(path, string(os.PathSeparator), "/")
			log.Trace().Strs("subtemplates", templates).Str("template", d.Name()).Msg("adding to bundle")

			t := template.Must(
				template.New(d.Name()).
					Funcs(sprig.TxtFuncMap()).
					Option("missingkey=error").
					ParseFS(tfs, templates...))
			b.Add(path, t)
		}
		return nil
	})

	return err
}

// Returns a bundle by traversing from templates/<bundleDir>
// Also traverses template-features/<feature> and adds it to the bundle
func NewBundle(bundleName string, features []string) (*Bundle, error) {
	var bfs fs.FS
	log.Logger = log.With().Str("bundle", bundleName).Logger()
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
		tree: &bundleNode{}}

	log.Debug().Msg("walking common tree")
	err := fsTreeWalk(b, bfs)
	if err != nil {
		return b, err
	}
	// Walk the features
	for _, feat := range features {
		var ffs fs.FS
		log.Logger = log.With().Str("feature", feat).Logger()
		if strings.HasPrefix(feat, ".") || strings.HasPrefix(feat, "/") {
			ffs = os.DirFS(feat)
		} else {
			var err error
			featPath := filepath.Join("template-features", bundleName, feat)
			fi, err := fs.Stat(Features, featPath)
			if err == nil && fi.IsDir() {
				ffs, err = fs.Sub(Features, featPath)
				if err != nil {
					log.Fatal().Err(err).Msgf("fetching embedded feature from %s", featPath)
				}
			} else {
				log.Fatal().Err(err).Msg("could not find embedded feature")
			}
		}
		err := fsTreeWalk(b, ffs)
		if err != nil {
			return b, err
		}		
	}
	
	return b, err
}
