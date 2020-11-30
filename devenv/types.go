package devenv

// DevEnv is a tyk env on the dev env. This is not a strict type because
// changes in repos lists will require a change in the type since this
// type would contain a list of repos. By using a map, we trade type
// checking of the state for flexibility in adding and removing repos.
type DevEnv map[string]interface{}

type baseError struct {
	Thing string
}

// NotFoundError is used to distinguish between other errors and this expected error
// in getEnv and elsewhere
type NotFoundError baseError

func (e NotFoundError) Error() string { return "does not exist: " + e.Thing }

// ExistsError is used when the environment exists but was updated via
// a method that is not idempotent
type ExistsError baseError

func (e ExistsError) Error() string { return "already exists: " + e.Thing }
