package models

import "time"

// Now returns the current time in the user's timezone.
func (u *User) Now() time.Time {
	return time.Now().In(u.Location())
}

// Location returns the user's timezone. Falls back to UTC
// if none is available.
func (u *User) Location() *time.Location {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}
	return loc
}
