package pebble

import "errors"

// Error definitions
var (
	ErrPebbleOpeningStore        = errors.New("error opening the store")
	ErrPebbleUnableToSetValue    = errors.New("unable to set value")
	ErrPebbleUnableToGetValue    = errors.New("unable to get value")
	ErrPebbleUnableToDeleteValue = errors.New("unable to delete value")
	ErrPebbleIteratingStore      = errors.New("unable to iterate over database")
	ErrPebbleClearingStore       = errors.New("unable to clear store")
	ErrPebbleUnableToBackup      = errors.New("unable to backup database")
	ErrPebbleUnableToRestore     = errors.New("unable to restore database")
	ErrPebbleClosingStore        = errors.New("unable to close database")
	ErrPebbleGettingStoreLength  = errors.New("unable to get database length")
)
