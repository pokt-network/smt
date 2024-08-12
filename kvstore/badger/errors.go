package badger

import (
	"errors"
)

var (
	// ErrBadgerOpeningStore is returned when the badger store cannot be opened
	// or an error occurs while opening/creating the BadgerKVStore
	ErrBadgerOpeningStore = errors.New("error opening the store")
	// ErrBadgerUnableToSetValue is returned when the badger store fails to
	// set a value
	ErrBadgerUnableToSetValue = errors.New("unable to set value")
	// ErrBadgerUnableToGetValue is returned when the badger store fails to
	// retrieve a value
	ErrBadgerUnableToGetValue = errors.New("unable to get value")
	// ErrBadgerUnableToDeleteValue is returned when the badger store fails to
	// delete a value
	ErrBadgerUnableToDeleteValue = errors.New("unable to delete value")
	// ErrBadgerIteratingStore is returned when the badger store fails to
	// iterate over the database
	ErrBadgerIteratingStore = errors.New("unable to iterate over database")
	// ErrBadgerClearingStore is returned when the badger store fails to
	// clear all values
	ErrBadgerClearingStore = errors.New("unable to clear store")
	// ErrBadgerUnableToBackup is returned when the badger store fails to
	// backup the database
	ErrBadgerUnableToBackup = errors.New("unable to backup database")
	// ErrBadgerUnableToRestore is returned when the badger store fails to
	// restore the database
	ErrBadgerUnableToRestore = errors.New("unable to restore database")
	// ErrBadgerClosingStore is returned when the badger store fails to
	// close the database
	ErrBadgerClosingStore = errors.New("unable to close database")
	// ErrBadgerGettingStoreLength is returned when the badger store fails to
	// get the length of the database
	ErrBadgerGettingStoreLength = errors.New("unable to get database length")
	// ErrBadgerUnableToCheckExistence is returned when the badger store fails to
	// check if a key exists
	ErrBadgerUnableToCheckExistence = errors.New("unable to check key existence")
)
