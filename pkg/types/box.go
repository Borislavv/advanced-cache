package types

type SizedBox[T any] struct {
	Value        T
	CalcWeightFn func(s *SizedBox[T]) int64
}

func (s *SizedBox[T]) Weight() int64 {
	return s.CalcWeightFn(s)
}
