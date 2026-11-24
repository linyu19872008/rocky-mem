// Package uid generates prefixed ULIDs (r_/m_/e_/…).
package uid

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

func Raw() string        { return "r_" + newULID() }
func Memory() string     { return "m_" + newULID() }
func Experience() string { return "e_" + newULID() }
func Conflict() string   { return "cg_" + newULID() }
func Request() string    { return "req_" + newULID() }
