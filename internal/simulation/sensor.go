package simulation

import (
	"fmt"
	"math/rand"
	"multilateration-sim/internal/common" // Замените на ваше имя модуля

	"github.com/google/uuid"
)

// NoiseFunction defines a function signature for adding noise to measurements.
// It takes the true distance and returns the noisy distance.
type NoiseFunction func(trueDistance float64) float64

// Sensor represents a sensor object in the simulation.
type Sensor struct {
	id              string
	position        common.Vector
	detectionRadius float64       // Maximum distance the sensor can detect
	noiseFunc       NoiseFunction // Function to add noise to measurements
	// Add other sensor-specific properties if needed
}

// NewSensor creates a new sensor at a given position.
func NewSensor(pos common.Vector, radius float64, noise NoiseFunction) *Sensor {
	// if noise == nil {
	// 	noise = func(d float64) float64 { return d } // Default: no noise
	// }
	// Если функция nil, остовляем nil. Для того что бы вывод (Sensor.String) корректно обробатывал такие случаи
	return &Sensor{
		id:              fmt.Sprintf("sensor-%s", uuid.NewString()[:8]),
		position:        pos.Clone(),
		detectionRadius: radius,
		noiseFunc:       noise,
	}
}

// GetID returns the unique identifier of the sensor.
func (s *Sensor) GetID() string {
	return s.id
}

// GetPosition returns the current position of the sensor.
func (s *Sensor) GetPosition() common.Vector {
	return s.position.Clone()
}

// SetPosition sets the position of the sensor.
func (s *Sensor) SetPosition(pos common.Vector) error {
	if pos.Dimension() != s.position.Dimension() {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", s.position.Dimension(), pos.Dimension())
	}
	s.position = pos.Clone()
	return nil
}

// Update for Sensor is currently empty as sensors are static in this version.
// If sensors could move, logic would go here.
func (s *Sensor) Update(deltaTime float64, bounds []float64) {
	// Sensors are static for now
}

// MeasureDistance measures the distance to a target object.
// Returns the measured distance (potentially with noise) and true if successful (within radius), false otherwise.
func (s *Sensor) MeasureDistance(target SimulationObject) (float64, bool, error) {
	targetPos := target.GetPosition()
	trueDist, err := s.position.Distance(targetPos)
	if err != nil {
		return 0, false, fmt.Errorf("error calculating distance for sensor %s: %w", s.id, err)
	}

	if s.detectionRadius > 0 && trueDist > s.detectionRadius {
		return 0, false, nil // Target is out of range
	}

	// Apply noise using the provided noise function
	var noisyDist float64
	if s.noiseFunc == nil {
		noisyDist = trueDist
	} else {
		noisyDist = s.noiseFunc(trueDist)
	}

	if noisyDist < 0 {
		noisyDist = 0 // Distance cannot be negative
	}

	return noisyDist, true, nil
}

// String representation for logging
func (s *Sensor) String() string {
	noiseDesc := "no"
	if s.noiseFunc != nil {
		// Basic check, won't work for complex closures but ok for now
		ptrVal := fmt.Sprintf("%p", s.noiseFunc)
		if ptrVal != fmt.Sprintf("%p", func(d float64) float64 { return d }) {
			noiseDesc = "yes"
		}
	}
	return fmt.Sprintf("Sensor[%s] Pos: %s Radius: %.2f Noise: %s", s.id, s.position, s.detectionRadius, noiseDesc)
}

// --- Example Noise Functions ---

// NoNoise is a NoiseFunction that adds no noise.
func NoNoise(trueDistance float64) float64 {
	return trueDistance
}

// GaussianNoise creates a NoiseFunction that adds Gaussian (normal) noise.
func GaussianNoise(stdDev float64) NoiseFunction {
	if stdDev < 0 {
		stdDev = 0
	}
	return func(trueDistance float64) float64 {
		noise := rand.NormFloat64() * stdDev
		return trueDistance + noise
	}
}

// UniformNoise creates a NoiseFunction that adds uniform noise within a range [-maxDelta, +maxDelta].
func UniformNoise(maxDelta float64) NoiseFunction {
	if maxDelta < 0 {
		maxDelta = 0
	}
	return func(trueDistance float64) float64 {
		noise := (rand.Float64()*2 - 1) * maxDelta // Noise between -maxDelta and +maxDelta
		return trueDistance + noise
	}
}

// PercentageNoise creates a NoiseFunction that adds noise as a percentage of the true distance.
// percentage is e.g., 0.05 for 5% noise. Noise is uniformly distributed within +/- percentage.
func PercentageNoise(percentage float64) NoiseFunction {
	if percentage < 0 {
		percentage = 0
	}
	return func(trueDistance float64) float64 {
		noiseMagnitude := trueDistance * percentage
		noise := (rand.Float64()*2 - 1) * noiseMagnitude // Noise between -noiseMagnitude and +noiseMagnitude
		return trueDistance + noise
	}
}

func (s *Sensor) DetectionRadius() float64 {
	return s.detectionRadius
}
