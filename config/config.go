package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Global vars that are available to all commands
// Loaded by loadConfig()
var ZoneID, Domain, TableName, RegistryID string
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
	} else if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		confPath = fmt.Sprintf("%s/%s", xdgHome, appName)
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
	ZoneID = viper.GetString("cluster.zoneid")
	Domain = viper.GetString("cluster.domain")

	log.Info().Interface("repos", Repos).Str("tablename", TableName).Str("registry", RegistryID).Str("zoneid", ZoneID).Str("domain", Domain).Msg("loaded environment")
}
