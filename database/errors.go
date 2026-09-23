package database

import "errors"

// ErrNotFound is returned by repository functions when a requested row either doesn't
// exist at all, or exists but isn't owned by the requesting user. Handlers should map
// this to an HTTP 404 - per the auth contract, a non-owner must never be able to tell
// the difference between "doesn't exist" and "exists but isn't yours".
var ErrNotFound = errors.New("not found")
