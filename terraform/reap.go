package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/TykTechnologies/gromit/devenv"
	"github.com/TykTechnologies/gromit/server"
	"github.com/TykTechnologies/gromit/util"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

// Reap is the entrypoint from the CLI
func Reap(confPath string) error {
	var e server.EnvConfig
	// Read env vars prefixed by GROMIT_
	err := envconfig.Process("gromit", &e)
	if err != nil {
		log.Fatal().Err(err).Msg("could not load env")
	}
	log.Info().Interface("env", e).Msg("loaded env")

	t := time.Now()
	defer func() {
		util.StatTime("reap.timetaken", time.Since(t))
	}()

	util.StatCount("reap.count", 1)
	if token := os.Getenv("TF_API_TOKEN"); len(token) > 0 {
		err = terraformCreds(token)
		if err != nil {
			util.StatCount("reap.failures", 1)
			log.Fatal().Err(err).Str("confPath", confPath).Msg("could not write tf creds")
		}
	}
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}

	db := dynamodb.New(cfg)
	envs, err := devenv.GetEnvsByState(db, e.TableName, devenv.DELETED, e.Repos)
	if err != nil {
		log.Fatal().Err(err).Str("state", devenv.DELETED).Str("table", e.TableName).Msg("could not get envs")
	}
	util.StatGauge("reap.nenvs", len(envs))

	var lastError error = nil
	for _, env := range envs {
		log.Info().Interface("env", env).Msg("deleting")
		envName := env[devenv.NAME].(string)
		envTime := time.Now()
		defer func() {
			util.StatTime(fmt.Sprintf("reap.%s.timetaken", envName), time.Since(envTime))
		}()

		err := os.RemoveAll(filepath.Join(confPath, envName))
		if err != nil {
			util.StatCount("reap.failures", 1)
			log.Error().Err(err).Str("env", envName).Msg("could not delete config tree")
			continue
		}
		// go.rice only works with string literals
		devManifest := rice.MustFindBox("devenv")
		tfDir, err := deployManifest(devManifest, envName)
		if err != nil {
			util.StatCount("reap.failures", 1)
			log.Error().Err(err).Msgf("could not deploy manifest for env %s", envName)
			lastError = err
			continue
		}
		err = makeInputVarfile(tfDir, env)
		if err != nil {
			util.StatCount("reap.failures", 1)
			log.Error().Err(err).Msgf("could not write input file for env %s", envName)
			lastError = err
			continue
		}
		gc, err := devenv.GetGromitCluster(envName)
		if err != nil {
			log.Error().Err(err).Str("env", envName).Msg("could not fetch cluster from ecs")
			util.StatCount("expose.failures", 1)
			lastError = err
			continue
		}
		err = gc.SyncDNS(route53.ChangeActionDelete, e.ZoneID, e.Domain)

		doTFCmd("destroy", envName, tfDir)
		err = devenv.DeleteEnv(db, e.TableName, envName)
		if err != nil {
			log.Error().Err(err).Str("env", envName).Msg("could not delete")
		}
	}
	return lastError
}
