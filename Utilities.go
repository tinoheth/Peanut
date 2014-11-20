package peanut

import (
	"os"
	"time"
)

type KWh float64

type ImpulseCount int64
type ImpulseSample struct {
	time.Time
	Impulses ImpulseCount
}
type ImpulseSampleData []ImpulseSample

type ImpulseData struct {
	*ImpulseSampleData
	ImpulseOffset ImpulseCount
	UnitFactor    Float
}

type Float float64
type FloatSample struct {
	time.Time
	Value Float
}
type FloatSampleData []FloatSample

func defaultTimeFactor() time.Duration {
	// Use time.Duration to get rid of casting
	//return 10000000 // If one second equals 100 time units,  we can measure nearly 11 minutes (6W) with uint16
	return time.Millisecond * 100 // If one second equals 10 time units, so we can measure 4 minutes (15W) with uint8

}

func DivideDuration(x time.Duration, y time.Duration) (result time.Duration) {
	result = x / y
	if (x-result*y)*2 >= y {
		result++
	}
	return
}

func NextWeekBreak(t time.Time) time.Time {
	y, m, d := t.Date()
	d += 7 - int(t.Weekday())
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	} else {
		return false
	}
}
