package vectorstore

import "math"

func DotProduct(a, b []float32) float32 {
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func NormalizeVector(v []float32) []float32 {
	var norm float64
	for _, val := range v {
		norm += float64(val) * float64(val)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return v
	}
	out := make([]float32, len(v))
	for i, val := range v {
		out[i] = float32(float64(val) / norm)
	}
	return out
}

func CosineSimilarity(a, b []float32) float32 {
	return DotProduct(NormalizeVector(a), NormalizeVector(b))
}
