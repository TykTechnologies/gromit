package orgs

import (
	"fmt"

	"github.com/TykTechnologies/gromit/util"
	"github.com/mongodb/mongo-tools-common/db"
	"github.com/mongodb/mongo-tools-common/options"
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
	opts.Direct = false
	opts.Namespace.DB = connOpts.Database

	err := opts.NormalizeOptionsAndURI()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot setup tooloptions")
	}
	return opts
}

func DumpFilteredCollections(uri *options.URI, queryField string, queryValue string, colls []string) error {
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
					Out:                    ".",
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

// RestoreCollections will restore all the collections in dir
func RestoreCollections(org string, uri *options.URI, dir string, dryRun bool) mongorestore.Result {
	connOpts := uri.ParsedConnString()
	topts := toolOpts(uri)
	mdb, err := mongorestore.New(mongorestore.Options{
		ToolOptions: topts,
		InputOptions: &mongorestore.InputOptions{
			//Objcheck: true,
			Directory: dir,
		},
		OutputOptions: &mongorestore.OutputOptions{
			Drop:                   false,
			DryRun:                 dryRun,
			StopOnError:            true,
			NumParallelCollections: 4,
		},
		TargetDirectory: dir,
		NSOptions: &mongorestore.NSOptions{
			NSInclude: []string{connOpts.Database + ".*"},
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("cannot instantiate mongorestore")
	}
	defer mdb.Close()

	return mdb.Restore()
}
