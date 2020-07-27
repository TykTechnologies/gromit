package devenv

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
