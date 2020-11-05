package server

import (
	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
)

// Serve starts the gromit server
func Serve(ca string, cert string, key string) {
	log.Info().Str("name", util.Name).Str("component", "serve").Str("version", util.Version).Msg("starting")
	a := App{}
	a.Init(ca)

	a.Run(":443", cert, key)
}
