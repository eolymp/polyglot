package oauth

import (
	"strings"
	"time"
)

type Token struct {
	ID         string    // unique identifier of the token
	Active     bool      // active status
	Restricted bool      // restricted status
	Expires    time.Time // expiration time
	Scopes     []string  // list of scopes
	Roles      []string  // list of roles
	Identity   *Identity // optionally, user identity assigned to the token
}

type Identity struct {
	UserID   string
	DeviceID string
	Username string
}

// String returns token string, it repeats ID() method, but has different semantic meaning, this method returns string
// representation of the token, while ID() returns token ID. Use this method if you want to print token as a string
// somewhere, use ID() when you need token string (eg. add it to the header).
func (t Token) String() string {
	return t.ID
}

// Valid returns true if token is valid
func (t Token) Valid() bool {
	return t.Active && time.Until(t.Expires) > 0
}

// Has scope(s), returns true if token has ALL scopes
func (t Token) Has(scopes ...string) bool {
	need := map[string]bool{}
	for _, scope := range scopes {
		need[strings.ToLower(scope)] = true
	}

	unmatched := len(need) // number of unique scopes requested

	for _, scope := range t.Scopes {
		scope := strings.ToLower(scope)
		if need[scope] {
			need[scope] = false // make sure we don't decrease unmatched twice for the same scope
			unmatched--
		}

		if unmatched == 0 {
			return true
		}
	}

	return unmatched == 0
}
