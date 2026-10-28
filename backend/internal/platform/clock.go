package platform

import "time"

// SystemClock is the production Clock. Times are always UTC.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
