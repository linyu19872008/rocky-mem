// Package conformance pins observable Store behaviour shared by every backend.
//
// SQLite runs these tests in-process today. PostgreSQL should call the same
// Suite once a postgres.Open exists — do not fork the assertions.
package conformance
