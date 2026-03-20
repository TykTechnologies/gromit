package policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// GenerateStatic generates static files for the TUI
func GenerateStatic(configDir, outDir string) error {
	av, err := loadAllVariations(configDir)
	if err != nil {
		return fmt.Errorf("loading variations from %s: %w", configDir, err)
	}

	for filename, v := range av {
		// Determine tsv name: strip .yml/.yaml and trailing s
		tsv := strings.TrimSuffix(filename, ".yaml")
		tsv = strings.TrimSuffix(tsv, ".yml")
		tsv = strings.TrimSuffix(tsv, "s")

		// Dump the entire variation
		dumpPath := filepath.Join(outDir, "v2", "dump", tsv)
		if err := writeJSON(dumpPath, v); err != nil {
			return err
		}

		for _, path := range v.Paths {
			repo := path.Repo
			branch := path.Branch
			trigger := path.Trigger
			ts := path.Testsuite

			m := v.Lookup(repo, branch, trigger, ts)
			if m == nil {
				continue
			}

			// Generate v2/{tsv}/{repo}/{branch}/{trigger}/{ts}.gho
			ghoPath := filepath.Join(outDir, "v2", tsv, repo, branch, trigger, ts+".gho")
			if err := writeGHO(ghoPath, *m); err != nil {
				return err
			}

			// Iterate over fields of ghMatrix
			val := reflect.ValueOf(*m)
			typ := val.Type()

			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				fieldValue := val.Field(i)
				jsonTag := field.Tag.Get("json")
				if jsonTag == "" || jsonTag == "-" {
					continue
				}
				if isEmpty(fieldValue) {
					continue
				}
				fieldJSONPath := filepath.Join(outDir, "v2", tsv, repo, branch, trigger, ts, jsonTag+".json")
				if err := writeJSON(fieldJSONPath, fieldValue.Interface()); err != nil {
					return err
				}

				// Generate v2/{tsv}/{repo}/{branch}/{trigger}/{ts}/{field}.gho
				fieldGHOPath := filepath.Join(outDir, "v2", tsv, repo, branch, trigger, ts, jsonTag+".gho")
				if err := writeFieldGHO(fieldGHOPath, jsonTag, fieldValue.Interface()); err != nil {
					return err
				}

				// Legacy v1 endpoint, only for prod-variation
				if tsv == "prod-variation" {
					// The legacy endpoint uses capitalized field names in the URL in some cases, but the instructions say {field}.
					// Wait, the test says: /api/repo1/br0/tr0/ts0/EnvFiles
					// Let's use the struct field name for legacy v1
					legacyPath := filepath.Join(outDir, "api", repo, branch, trigger, ts, field.Name)
					if err := writeJSON(legacyPath, fieldValue.Interface()); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func writeJSON(path string, obj any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeGHO(path string, obj any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	val := reflect.ValueOf(obj)
	typ := val.Type()

	var buf bytes.Buffer
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		fieldName := field.Tag.Get("json")
		if fieldName == "" || fieldName == "-" {
			continue
		}
		if isEmpty(fieldValue) {
			continue
		}
		fjson, err := json.Marshal(fieldValue.Interface())
		if err != nil {
			return err
		}
		buf.WriteString(fieldName + "<<EOF\n")
		buf.Write(fjson)
		buf.WriteString("\nEOF\n")
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func writeFieldGHO(path string, fieldName string, obj any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var buf bytes.Buffer
	fjson, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Struct {
		typ := val.Type()
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			fv := val.Field(i)
			fn := f.Tag.Get("json")
			if fn == "" || fn == "-" {
				continue
			}
			if isEmpty(fv) {
				continue
			}
			fj, err := json.Marshal(fv.Interface())
			if err != nil {
				return err
			}
			buf.WriteString(fn + "<<EOF\n")
			buf.Write(fj)
			buf.WriteString("\nEOF\n")
		}
	} else {
		buf.WriteString(fieldName + "<<EOF\n")
		buf.Write(fjson)
		buf.WriteString("\nEOF\n")
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.String, reflect.Array:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isEmpty(v.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return true
		}
		return isEmpty(v.Elem())
	default:
		return v.IsZero()
	}
}