package time

import (
	goTime "time"
)

type (
	Duration = goTime.Duration
	Time     = goTime.Time
	Timer    = goTime.Timer
)

const (
	Nanosecond  = goTime.Nanosecond
	Microsecond = goTime.Microsecond
	Millisecond = goTime.Millisecond
	Second      = goTime.Second
	Minute      = goTime.Minute
	Hour        = goTime.Hour

	April  = goTime.April
	August = goTime.August

	RFC3339  = goTime.RFC3339
	RFC1123  = goTime.RFC1123
	DateTime = goTime.DateTime
)

var (
	UTC   = goTime.UTC
	Local = goTime.Local
)

func Now() Time {
	return goTime.Now().UTC()
}

func Date(year int, month int, day int, hour int, min int, sec int, nsec int, loc *goTime.Location) Time {
	return goTime.Date(year, goTime.Month(month), day, hour, min, sec, nsec, loc)
}

func Unix(sec int64, nsec int64) Time {
	return goTime.Unix(sec, nsec).UTC()
}

func UnixMilli(ms int64) Time {
	return goTime.UnixMilli(ms).UTC()
}

func UnixMicro(micro int64) Time {
	return goTime.UnixMicro(micro).UTC()
}

func UnixNano(ns int64) Time {
	return goTime.Unix(0, ns).UTC()
}

func ParseDuration(d string) (Duration, error) {
	return goTime.ParseDuration(d)
}

func NewTicker(d Duration) *goTime.Ticker {
	return goTime.NewTicker(d)
}

func NewTimer(d Duration) *goTime.Timer {
	return goTime.NewTimer(d)
}

func After(d Duration) <-chan Time {
	return goTime.After(d)
}

func Since(t Time) Duration {
	return goTime.Since(t)
}

func Parse(layout, value string) (Time, error) {
	return goTime.Parse(layout, value)
}

func Sleep(d Duration) {
	goTime.Sleep(d)
}
