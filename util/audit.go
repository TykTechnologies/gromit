package util

import (
	"os"

	"github.com/rs/zerolog"
)

var auditLog = zerolog.New(os.Stderr).With().Timestamp().Logger()
var envLog = zerolog.New(os.Stderr).With().Timestamp().Logger()
