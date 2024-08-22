package pebble

import "errors"

// Error definitions
var (
	// ErrPebbleOpeningStore is returned when there's an error opening the Pebble database.
	ErrPebbleOpeningStore = errors.New("error opening the store")

	// ErrPebbleUnableToSetValue is returned when a value cannot be set in the store.
	ErrPebbleUnableToSetValue = errors.New("unable to set value")

	// ErrPebbleUnableToGetValue is returned when a value cannot be retrieved from the store.
	ErrPebbleUnableToGetValue = errors.New("unable to get value")

	// ErrPebbleUnableToDeleteValue is returned when a value cannot be deleted from the store.
	ErrPebbleUnableToDeleteValue = errors.New("unable to delete value")

	// ErrPebbleIteratingStore is returned when there's an error iterating over the database.
	ErrPebbleIteratingStore = errors.New("unable to iterate over database")

	// ErrPebbleClearingStore is returned when there's an error clearing all data from the store.
	ErrPebbleClearingStore = errors.New("unable to clear store")

	// ErrPebbleUnableToBackup is returned when there's an error backing up the database.
	ErrPebbleUnableToBackup = errors.New("unable to backup database")

	// ErrPebbleUnableToRestore is returned when there's an error restoring the database from a backup.
	ErrPebbleUnableToRestore = errors.New("unable to restore database")

	// ErrPebbleClosingStore is returned when there's an error closing the database connection.
	ErrPebbleClosingStore = errors.New("unable to close database")

	// ErrPebbleGettingStoreLength is returned when there's an error getting the number of key-value pairs in the database.
	ErrPebbleGettingStoreLength = errors.New("unable to get database length")
)
