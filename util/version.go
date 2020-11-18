package util

// Populated by -ldflags -X
var (
	version   = "known at build time"
	commit    = "known at build time"
	buildDate = "known at build time"
	name      = "Gromit"
)

func Version() string {
	return version
}

func Commit() string {
	return commit + " built on " + buildDate
}

func Name() string {
	return name
}
