package common

import (
	"fmt"
	"strings"
	"time"
)

// UnixMillisToNano converts Unix milli time to UnixNano
func UnixMillisToNano(milli int64) int64 {
	return milli * int64(time.Millisecond)
}

// UnixMillis converts a UnixNano timestamp to milliseconds
func UnixMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// RecvWindow converts a supplied time.Duration to milliseconds
func RecvWindow(d time.Duration) int64 {
	return int64(d) / int64(time.Millisecond)
}

// TimeFromUnixTimestampFloat format
func TimeFromUnixTimestampFloat(raw interface{}) (time.Time, error) {
	ts, ok := raw.(float64)
	if !ok {
		return time.Time{}, fmt.Errorf("unable to parse, value not float64: %T", raw)
	}
	return time.Unix(0, int64(ts)*int64(time.Millisecond)), nil
}

// ConvertTimeStringToRFC3339 converts returned time string to time.Time
func ConvertTimeStringToRFC3339(timestamp string) (time.Time, error) {
	split := strings.Split(timestamp, " ")
	join := strings.Join(split, "T")
	return time.Parse(time.RFC3339, join+"Z")
}
