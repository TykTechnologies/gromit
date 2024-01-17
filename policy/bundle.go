package policy

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/yamlfmt/formatters/basic"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Files common to all features
//
//go:embed templates all:templates
var templates embed.FS

// bundleNode of a directory tree
type bundleNode struct {
	Name     string
	path     string
	template *template.Template
	Children []*bundleNode
}

// Bundle represents a directory tree, instantiated by NewBundle()
type Bundle struct {
	Name    string
	tree    *bundleNode
	vdrMap  validatorMap
	yamlfmt *basic.BasicFormatter
	isYaml  *regexp.Regexp
}

// getSubTemplates returns a list of subtemplate definitions that are available to all templates
func getSubTemplates(subfs fs.FS, root string) ([]string, error) {
	var stList []string
	err := fs.WalkDir(subfs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			stList = append(stList, path)
		}
		return nil
	})
	return stList, err
}

// Add adds the path and corresponding template into the templateNode tree
// This code due to ChatGPT
func (b *Bundle) Add(path string, template *template.Template) {
	// Split the path into its components and drop leading template/<bundle>
	components := strings.Split(path, string(os.PathSeparator))[2:]
	path = filepath.Join(components...)

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
	newNode := &bundleNode{Name: components[len(components)-1], path: path, template: template}
	parent.Children = append(parent.Children, newNode)
}

// Render will walk a tree given in n, depth first, rendering leaves
// bv will accept any type which will used directly to render the
// templates
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
		var buf bytes.Buffer
		if err := n.template.Execute(&buf, bv); err != nil {
			return nil, fmt.Errorf("rendering %s: %v", n.Name, err)
		}
		var opFile = filepath.Join(opDir, n.path)
		if err := b.write(&buf, opFile); err != nil {
			return nil, err
		}
		// Make all *.sh files executable
		if filepath.Ext(opFile) == ".sh" {
			log.Info().Msgf(".sh file", opFile)
			err := os.Chmod(opFile, 0775)
			if err != nil {
				return nil, err
			}
		}
		renderedFiles = append(renderedFiles, n.path)
	}
	return renderedFiles, nil
}

// writes runs a validator (if it can) and yamlfmt the rendered output
// before writing it out
func (b *Bundle) write(buf *bytes.Buffer, opFile string) error {
	vdr := getValidator(strings.Split(opFile, string(os.PathSeparator)))
	if vdr != UNKNOWN_VALIDATOR {
		var y any
		if err := yaml.Unmarshal(buf.Bytes(), &y); err != nil {
			return fmt.Errorf("could not unmarshal %s: %v", opFile, err)
		}
		if err := b.vdrMap[vdr].Validate(y); err != nil {
			return fmt.Errorf("%s failed validation: %#v", opFile, err)
		}
	}
	var op []byte
	var err error
	if b.isYaml.MatchString(opFile) {
		op, err = b.yamlfmt.Format(buf.Bytes())
		if err != nil {
			os.WriteFile("error.yaml", buf.Bytes(), 0644)
			return fmt.Errorf("could not yamlfmt %s: %#v", opFile, err)
		}
	} else {
		op = buf.Bytes()
	}
	dir, _ := filepath.Split(opFile)
	err = os.MkdirAll(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkdirall %s: %v", dir, err)
	}
	opf, err := os.Create(opFile)
	if err != nil {
		return fmt.Errorf("create %s: %v", opFile, err)
	}
	defer opf.Close()
	_, err = opf.Write(op)
	return err
}

// String will provide a human readable bundle listing
func (b *Bundle) String() string {
	return fmt.Sprintf(b.Name) + b.tree.String(0)
}

// Count is the public function that wraps the implementation
func (b *Bundle) Count() int {
	return b.tree.Count(b.tree)
}

// Print an indented tree
func (n *bundleNode) String(indent int) string {
	op := fmt.Sprintf("%s%s\n", strings.Repeat("  ", indent), n.Name)
	for _, child := range n.Children {
		op += child.String(indent + 1)
	}
	return op
}

// Count return the number of leaf nodes, which is a count of the
// files that will be rendered, thanks ChatGPT 3.5(turbo)
func (n *bundleNode) Count(bn *bundleNode) int {
	count := 0
	if len(bn.Children) == 0 {
		return 1
	}
	for _, child := range bn.Children {
		count += n.Count(child)
	}
	return count
}

// fsTreeWalk will walk the complete tree of tfs and add templates to the supplied Bundle b.
// Used to walk both the common bundle and the features bundle
func fsTreeWalk(b *Bundle, tfs fs.FS, root string, subTemps []string) error {
	err := fs.WalkDir(tfs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root node
		if path == "." {
			return nil
		}
		// Do not recurse into sub-templates
		if d.IsDir() && strings.HasSuffix(path, ".d") {
			return fs.SkipDir
		}
		if !d.IsDir() {
			// The top-level template must be the first element
			subTemps = append([]string{path}, subTemps...)

			stPath := path + ".d"
			fi, err := fs.Stat(tfs, stPath)
			if err == nil && fi.IsDir() {
				des, err := fs.ReadDir(tfs, stPath)
				if err != nil {
					return err
				}
				for _, de := range des {
					subTemps = append(subTemps, filepath.Join(stPath, de.Name()))
				}
			}
			// Normalize the path to use '/' as the separator
			path = strings.ReplaceAll(path, string(os.PathSeparator), "/")
			log.Trace().Strs("subtemplates", subTemps).Str("template", d.Name()).Msg("adding to bundle")

			t := template.Must(
				template.New(d.Name()).
					Funcs(sprig.TxtFuncMap()).
					Option("missingkey=error").
					ParseFS(tfs, subTemps...))
			b.Add(path, t)
		}
		return nil
	})

	return err
}

// Returns a bundle by walking templates/<features>
func NewBundle(features []string) (*Bundle, error) {
	var err error
	vdrMap, err := loadValidators()
	if err != nil {
		log.Warn().Err(err).Msg("loading validators")
	}
	config := basic.DefaultConfig()
	config.ScanFoldedAsLiteral = true
	b := &Bundle{
		Name:   strings.Join(features, "-"),
		tree:   &bundleNode{},
		vdrMap: vdrMap,
		yamlfmt: &basic.BasicFormatter{
			Config:   config,
			Features: basic.ConfigureFeaturesFromConfig(config),
		},
		isYaml: regexp.MustCompile("\\.y(a)?ml$"),
	}
	log.Logger = log.With().Strs("features", features).Logger()
	stList, err := getSubTemplates(templates, filepath.Join("templates", "subtemplates"))
	if err != nil {
		log.Fatal().Err(err).Msg("walking subtemplates")
	}
	for _, feat := range features {
		featPath := filepath.Join("templates", feat)
		err = fsTreeWalk(b, templates, featPath, stList)
		if err != nil {
			if os.IsNotExist(err) {
				log.Debug().Msgf("did not find bundle for feature %s, assuming it does not have any files.", feat)
				err = nil
			} else {
				log.Fatal().Err(err).Msgf("walking feature %s", feat)
			}
		}
	}
	return b, err
}
