package orgs

import (
	"github.com/mongodb/mongo-tools-common/options"
)

// ParseMongoURI can parse URLs like mongodb:///...
func ParseMongoURI(url string) (*options.URI, error) {
	return options.NewURI(url)
}
