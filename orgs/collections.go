package orgs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TykTechnologies/gromit/util"
	"github.com/mongodb/mongo-tools-common/db"
	"github.com/mongodb/mongo-tools-common/options"
	"github.com/mongodb/mongo-tools-common/signals"
	"github.com/mongodb/mongo-tools/mongodump"
	"github.com/mongodb/mongo-tools/mongorestore"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// ParseMongoURI can parse URLs like mongodb:///...
func ParseMongoURI(url string) (*options.URI, error) {
	return options.NewURI(url)
}

func toolOpts(uri *options.URI) *options.ToolOptions {
	opts := options.New(util.Name, util.Version, util.Commit, "see gromit help", false, options.EnabledOptions{Auth: true, Connection: true, Namespace: true, URI: true})
	connOpts := uri.ParsedConnString()
	opts.URI = uri
	opts.Namespace.DB = connOpts.Database
	// opts.SSL = &options.SSL{}
	// opts.SSL.UseSSL = connOpts.SSL
	opts.ReplicaSetName = connOpts.ReplicaSet

	err := opts.NormalizeOptionsAndURI()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot setup tooloptions")
	}
	return opts
}

// FastFilteredCollections dumps collections concurrently
func FastFilteredCollections(uri *options.URI, queryField string, queryValue string, colls []string) error {
	mongoGrp := new(errgroup.Group)
	for _, coll := range colls {
		coll := coll // https://golang.org/doc/faq#closures_and_goroutines
		mongoGrp.Go(func() error {
			topts := toolOpts(uri)
			sp, err := db.NewSessionProvider(*topts)
			if err != nil {
				return err
			}
			topts.Namespace.Collection = coll
			mdb := mongodump.MongoDump{
				ToolOptions: topts,
				InputOptions: &mongodump.InputOptions{
					Query: fmt.Sprintf(`{"%s": "%s"}`, queryField, queryValue),
				},
				OutputOptions: &mongodump.OutputOptions{
					Out:                    queryValue,
					NumParallelCollections: 4,
				},
				SessionProvider: sp,
			}
			err = mdb.Init()
			if err != nil {
				return err
			}
			err = mdb.Dump()
			if err != nil {
				return err
			}
			return nil
		})
	}
	err := mongoGrp.Wait()
	if err != nil {
		return err
	}
	return nil
}

// SlowFilteredCollections dumps collections sequentially
func SlowFilteredCollections(uri *options.URI, queryField string, queryValue string, colls []string) error {
	for _, coll := range colls {
		coll := coll // https://golang.org/doc/faq#closures_and_goroutines
		topts := toolOpts(uri)
		sp, err := db.NewSessionProvider(*topts)
		if err != nil {
			return err
		}
		topts.Namespace.Collection = coll
		mdb := mongodump.MongoDump{
			ToolOptions: topts,
			InputOptions: &mongodump.InputOptions{
				Query: fmt.Sprintf(`{"%s": "%s"}`, queryField, queryValue),
			},
			OutputOptions: &mongodump.OutputOptions{
				Out:                    queryValue,
				NumParallelCollections: 4,
			},
			SessionProvider: sp,
		}
		err = mdb.Init()
		if err != nil {
			return err
		}
		err = mdb.Dump()
		if err != nil {
			return err
		}
	}
	return nil
}

func DumpAnalyticzCollections(uri *options.URI, org string, colls []string) error {
	for _, coll := range colls {
		topts := toolOpts(uri)
		sp, err := db.NewSessionProvider(*topts)
		if err != nil {
			return err
		}
		topts.Namespace.Collection = coll
		mdb := mongodump.MongoDump{
			ToolOptions:  topts,
			InputOptions: &mongodump.InputOptions{},
			OutputOptions: &mongodump.OutputOptions{
				Out:                    org,
				NumParallelCollections: 4,
			},
			SessionProvider: sp,
		}
		err = mdb.Init()
		if err != nil {
			return err
		}
		err = mdb.Dump()
		if err != nil {
			return err
		}
	}
	return nil
}

type collection struct {
	Name     string
	BsonFile string
}

func (c *collection) restore(uri *options.URI, dryRun bool) mongorestore.Result {
	opts := toolOpts(uri)

	nsOpts := &mongorestore.NSOptions{
		NSInclude: []string{opts.DB + ".*"},
	}
	opts.AddOptions(nsOpts)
	inputOpts := &mongorestore.InputOptions{
		Objcheck: true,
		//Directory: dir,
	}
	opts.AddOptions(inputOpts)
	outputOpts := &mongorestore.OutputOptions{
		Drop:                   false,
		DryRun:                 dryRun,
		StopOnError:            true,
		NumParallelCollections: 4,
	}
	opts.AddOptions(outputOpts)

	sp, err := db.NewSessionProvider(*opts)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create session provider")
	}

	mdb := mongorestore.MongoRestore{
		ToolOptions:     opts,
		OutputOptions:   outputOpts,
		InputOptions:    inputOpts,
		NSOptions:       nsOpts,
		TargetDirectory: c.BsonFile,
		SessionProvider: sp,
	}
	err = mdb.ParseAndValidateOptions()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid options")
	}

	finishedChan := signals.HandleWithInterrupt(mdb.HandleInterrupt)
	defer close(finishedChan)

	return mdb.Restore()
}

// findCollections will walk the tree from dir and update the colls pointer
// with the collections that it finds
func findCollections(dir string, colls *[]collection) error {
	rootDir, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer rootDir.Close()
	entries, err := rootDir.Readdir(0)
	if err != nil {
		return err
	}

	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if e.IsDir() {
			findCollections(path, colls)
		} else {
			if strings.HasSuffix(path, ".bson") {
				log.Debug().Str("path", path).Msg("found")
				c := collection{
					Name:     strings.TrimSuffix(e.Name(), ".bson"),
					BsonFile: path,
				}
				*colls = append(*colls, c)
			}
		}
	}
	return nil
}

// RestoreCollections will restore all the collections in dir
func RestoreCollections(dir string, uri *options.URI, dryRun bool) []mongorestore.Result {
	var colls []collection

	err := findCollections(dir, &colls)
	if err != nil {
		log.Fatal().Err(err).Msg("finding collections")
	}
	log.Info().Interface("collections", colls).Msg("found")

	var results []mongorestore.Result
	for _, c := range colls {
		results = append(results, c.restore(uri, dryRun))
	}
	return results
}
