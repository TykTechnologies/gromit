package cmd

// exit codes from 10 onwards are used for custom
// exit codes on command errors.
const (
	exitLockAlreadyTaken = 10 + iota
	exitInvalidArguments
)
