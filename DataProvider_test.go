package peanut

import (
	"time"
)

import "testing"

func Equals(self []byte, other []byte) bool {
	if len(self) != len(other) {
		return false
	} else {
		for i, value := range self {
			if value != other[i] {
				return false
			}
		}
		return true
	}
}

func TestRawConversion(t *testing.T) {
	input := []byte{1, 2, 3, 3, 2, 1}
	var valueDelta float64 = 1 / 1000
	startTime := time.Now()
	intermediate := translateFromRaw(input, startTime, defaultTimeFactor(), 0, valueDelta)
	result := translateToRaw(*intermediate, startTime, defaultTimeFactor(), valueDelta)

	if Equals(input, result) {
		t.Error("Result")
	}
}
