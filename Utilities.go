package Peanut

import "time"

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
	return 10000000
	// One second equals 100 time units, so we can measure nearly 11 minutes (6W)
}

func DivideDuration(x time.Duration, y time.Duration) (result time.Duration) {
	result = x / y
	if (x-result*y)*2 >= y {
		result++
	}
	return
}
