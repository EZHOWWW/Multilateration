package common

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

// Vector represents a point or vector in n-dimensional space.
type Vector []float64

// NewVector creates a new vector of a given dimension.
func NewVector(dimension int) Vector {
	return make(Vector, dimension)
}

// NewRandomVector creates a vector with random coordinates within given bounds.
// bounds should have dimension * 2 elements: [minX, maxX, minY, maxY, ...]
func NewRandomVector(dimension int, bounds []float64) (Vector, error) {
	if len(bounds) != dimension*2 {
		return nil, fmt.Errorf("bounds length must be dimension * 2, got %d, expected %d", len(bounds), dimension*2)
	}
	v := NewVector(dimension)
	for i := 0; i < dimension; i++ {
		min := bounds[i*2]
		max := bounds[i*2+1]
		v[i] = min + rand.Float64()*(max-min) // Generate random float between min and max
	}
	return v, nil
}

// Dimension returns the dimension of the vector.
func (v Vector) Dimension() int {
	return len(v)
}

// Distance calculates the Euclidean distance between two vectors.
func (v Vector) Distance(other Vector) (float64, error) {
	if v.Dimension() != other.Dimension() {
		return 0, fmt.Errorf("vectors must have the same dimension: %d != %d", v.Dimension(), other.Dimension())
	}
	sumOfSquares := 0.0
	for i := range v {
		diff := v[i] - other[i]
		sumOfSquares += diff * diff
	}
	return math.Sqrt(sumOfSquares), nil
}

// Add adds another vector to this vector.
func (v Vector) Add(other Vector) (Vector, error) {
	if v.Dimension() != other.Dimension() {
		return nil, fmt.Errorf("vectors must have the same dimension: %d != %d", v.Dimension(), other.Dimension())
	}
	result := NewVector(v.Dimension())
	for i := range v {
		result[i] = v[i] + other[i]
	}
	return result, nil
}

// Subtract subtracts another vector from this vector.
func (v Vector) Subtract(other Vector) (Vector, error) {
	if v.Dimension() != other.Dimension() {
		return nil, fmt.Errorf("vectors must have the same dimension: %d != %d", v.Dimension(), other.Dimension())
	}
	result := NewVector(v.Dimension())
	for i := range v {
		result[i] = v[i] - other[i]
	}
	return result, nil
}

// MultiplyByScalar multiplies the vector by a scalar value.
func (v Vector) MultiplyByScalar(scalar float64) Vector {
	result := NewVector(v.Dimension())
	for i := range v {
		result[i] = v[i] * scalar
	}
	return result
}

// String returns a string representation of the vector.
func (v Vector) String() string {
	// Format with limited precision for cleaner output
	strs := make([]string, len(v))
	for i, val := range v {
		strs[i] = fmt.Sprintf("%.3f", val)
	}
	return fmt.Sprintf("[%s]", strings.Join(strs, ", ")) // Changed from %v for better formatting
}

// Clone creates a deep copy of the vector.
func (v Vector) Clone() Vector {
	clone := make(Vector, len(v))
	copy(clone, v)
	return clone
}

// NormSq calculates the squared Euclidean norm (magnitude squared) of the vector (dot product with itself).
func (v Vector) NormSq() float64 {
	sumOfSquares := 0.0
	for _, val := range v {
		sumOfSquares += val * val
	}
	return sumOfSquares
}

// --- Potentially add more vector operations as needed ---
// Magnitude (Norm), Normalize, DotProduct etc.

// --- Need strings import for String() method ---
