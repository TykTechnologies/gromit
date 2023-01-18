package config

import (
	"bytes"
	"os"
	"strings"

	_ "embed"

	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Global vars that are available to appropriate commands
// Loaded by loadConfig()
var ZoneID, Domain, TableName, RegistryID, RepoURLPrefix string
var Repos []string

//go:embed config.yaml
var config []byte

// LoadConfig is a helper function that loads the environment into the
// global variables TableName, RegistryID and so on defined at the top
// of this file. It is called from initConfig as well as any tests that
// need it
func LoadConfig(cfgFile string) {
	appName := util.Name()
	// Use the passed config file if it exists.
	viper.SetConfigFile(cfgFile)

	// If a config file is found, read it in.
	// Use the embedded config otherwise
	if err := viper.ReadInConfig(); err == nil {
		log.Debug().Str("file", viper.ConfigFileUsed()).Msg("reading config from, use env vars to override specific parameters")
	} else {
		log.Debug().Err(err).Msg("Error parsing config file, using embedded config")
		if err = viper.ReadConfig(bytes.NewReader(config)); err != nil {
			log.Fatal().Bytes("config", config).Msg("could not read embedded config")
		} else {
			log.Debug().Msg("using embedded config, use env vars to override")
		}
	}
	// Look in env first for every viper.Get* call
	viper.AutomaticEnv()
	viper.SetEnvPrefix(strings.ToUpper(appName))
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	Repos = strings.Split(viper.GetString("repos"), ",")
	RegistryID = viper.GetString("registryid")
	TableName = viper.GetString("tablename")

	// Setup logging as per config file, overriding the command line options
	if logLevel := viper.GetString("loglevel"); logLevel != "" {
		ll, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			log.Warn().Str("level", logLevel).Msg("Could not parse, defaulting to debug.")
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(ll)
		}
	}
	if viper.GetBool("textlogs") {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Debug().Interface("repos", Repos).Str("tablename", TableName).Str("registry", RegistryID).Str("file", viper.ConfigFileUsed()).Msg("loaded config from file")
}

// LoadClusterConfig loads the config that the cluster command will need
func LoadClusterConfig() {
	ZoneID = viper.GetString("cluster.zoneid")
	Domain = viper.GetString("cluster.domain")
	log.Info().Str("zoneid", ZoneID).Str("domain", Domain).Msg("loaded cluster config")
}
