package models

import "time"

// Now returns the current time in the user's timezone.
func (u *User) Now() time.Time {
	return time.Now().In(u.Location())
}

// Location returns the user's timezone. Falls back to UTC
// if none is available.
func (u *User) Location() *time.Location {
	return LocationFromTimezone(u.Timezone)
}

func LocationFromTimezone(timezone string) *time.Location {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}
