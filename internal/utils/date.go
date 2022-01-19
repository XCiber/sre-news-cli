package utils

import "time"

func StartTimeOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func EndTimeOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 59, t.Location())
}

func NasdaqOpeningTimeOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 14, 30, 00, 00, t.Location())
}
