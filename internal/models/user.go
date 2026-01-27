package models

import "time"

// Now returns the current time in the user's timezone.
// Falls back to UTC if the timezone is invalid.
func (u *User) Now() time.Time {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc)
}
