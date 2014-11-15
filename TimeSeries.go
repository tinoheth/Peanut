// TimeSeries
package Peanut

func Derive(s []ImpulseSample, factor Float) []FloatSample {
	i := len(s)
	if i > 0 {
		i--
	}
	result := make([]FloatSample, i)
	for i > 0 {
		current := s[i]
		i--
		prev := s[i]
		v := Float(current.Impulses-prev.Impulses) / factor / Float(current.Sub(prev.Time).Seconds())
		result[i] = FloatSample{prev.Time, v}
	}
	return result
}
