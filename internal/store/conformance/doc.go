// Package conformance pins observable Store behaviour shared by every backend.
//
// SQLite runs in-process. PostgreSQL runs the same Suite when
// ROCKYMEM_TEST_PG_DSN is set — do not fork the assertions.
package conformance
