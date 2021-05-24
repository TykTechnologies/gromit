package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TykTechnologies/gromit/policy"
	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Global vars that are available to appropriate commands
// Loaded by loadConfig()
var ZoneID, Domain, TableName, RegistryID, Branch, RepoURLPrefix string
var Repos []string

// LoadConfig is a helper function that loads the environment into the
// global variables TableName, RegistryID and so on defined at the top
// of this file. It is called from initConfig as well as any tests that
// need it
func LoadConfig(cfgFile string) {
	appName := util.Name()
	viper.AddConfigPath(fmt.Sprintf("/conf/%s", appName))
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Local config path
	var confPath = fmt.Sprintf("%s/.config/%s", os.Getenv("HOME"), appName)
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		confPath = filepath.Dir(cfgFile)
		log.Debug().Str("file", cfgFile).Msg("using config file")
	} else if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		confPath = filepath.Join(xdgHome, appName)
		log.Debug().Str("path", confPath).Msg("looking for config path")
	} else {
		log.Debug().Str("path", confPath).Msg("using default config path")
	}

	viper.AddConfigPath(confPath)
	viper.Set("confpath", confPath)
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debug().Str("file", viper.ConfigFileUsed()).Msg("reading config from")
	} else {
		log.Debug().Msg("No config file read, depending on env variables")
	}
	// Look in env first for every viper.Get* call
	viper.AutomaticEnv()
	viper.SetEnvPrefix("GROMIT")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	Repos = strings.Split(viper.GetString("repos"), ",")
	RegistryID = viper.GetString("registryid")
	TableName = viper.GetString("tablename")

	// Setup logging as per config file, overriding the command line options
	logLevel := viper.GetString("loglevel")
	ll, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Warn().Str("level", logLevel).Msg("Could not parse, defaulting to debug.")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(ll)
	}
	if viper.GetBool("textlogs") {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Interface("repos", Repos).Str("tablename", TableName).Str("registry", RegistryID).Str("file", viper.ConfigFileUsed()).Msg("loaded config from file")
}

// LoadClusterConfig loads the config that the cluster command will need
func LoadClusterConfig() {
	ZoneID = viper.GetString("cluster.zoneid")
	Domain = viper.GetString("cluster.domain")
	log.Info().Str("zoneid", ZoneID).Str("domain", Domain).Msg("loaded cluster config")
}

// GetPolicyConfig returns the policies as a map of repos to policies
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *policy.RepoPolicies) error {
	log.Info().Msg("loading repo policies")
	return viper.UnmarshalKey("policy", policies)
}
