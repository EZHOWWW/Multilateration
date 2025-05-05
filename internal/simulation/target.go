package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"multilateration-sim/internal/common" // Замените на ваше имя модуля
	"time"

	"github.com/google/uuid" // Для генерации уникальных ID
)

// Target represents a target object in the simulation.
type Target struct {
	id       string
	position common.Vector
	velocity common.Vector // Current velocity for movement
	// Add other target-specific properties if needed
}

// NewTarget creates a new target at a given position.
func NewTarget(pos common.Vector) *Target {
	dim := pos.Dimension()
	// Start with zero velocity initially
	vel := common.NewVector(dim)
	return &Target{
		id:       fmt.Sprintf("target-%s", uuid.NewString()[:8]), // Shorter unique ID
		position: pos.Clone(),                                    // Clone to avoid external modification
		velocity: vel,
	}
}

// GetID returns the unique identifier of the target.
func (t *Target) GetID() string {
	return t.id
}

// GetPosition returns the current position of the target.
func (t *Target) GetPosition() common.Vector {
	// Return a clone to prevent modification of the internal state
	return t.position.Clone()
}

// SetPosition sets the position of the target.
func (t *Target) SetPosition(pos common.Vector) error {
	if pos.Dimension() != t.position.Dimension() {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", t.position.Dimension(), pos.Dimension())
	}
	t.position = pos.Clone() // Store a clone
	return nil
}

// Update implements the random walk movement and boundary checks.
func (t *Target) Update(deltaTime float64, bounds []float64) {
	dim := t.position.Dimension()
	if len(bounds) != dim*2 {
		fmt.Printf("Warning: Target %s received invalid bounds length\n", t.id)
		return // Or handle error more gracefully
	}

	// --- Simple Random Walk Logic ---
	// Adjust velocity slightly randomly
	accelerationScale := 5.0 // How much velocity can change per second
	for i := 0; i < dim; i++ {
		// Add a small random change to velocity
		t.velocity[i] += (rand.Float64()*2 - 1) * accelerationScale * deltaTime
	}

	// --- Limit Velocity (Optional) ---
	maxSpeed := 10.0 // Maximum units per second
	currentSpeedSq := 0.0
	for _, v := range t.velocity {
		currentSpeedSq += v * v
	}
	if currentSpeedSq > maxSpeed*maxSpeed {
		scale := maxSpeed / math.Sqrt(currentSpeedSq)
		t.velocity = t.velocity.MultiplyByScalar(scale)
	}

	// --- Update Position ---
	deltaPos := t.velocity.MultiplyByScalar(deltaTime)
	newPos, err := t.position.Add(deltaPos)
	if err != nil {
		fmt.Printf("Error updating target %s position: %v\n", t.id, err)
		return // Skip update if dimensions mismatch (shouldn't happen here)
	}

	// --- Boundary Collision Check (Bounce) ---
	for i := 0; i < dim; i++ {
		minBound := bounds[i*2]
		maxBound := bounds[i*2+1]
		if newPos[i] < minBound {
			newPos[i] = minBound + (minBound - newPos[i]) // Reflect position
			t.velocity[i] *= -0.8                         // Reverse and dampen velocity component
		} else if newPos[i] > maxBound {
			newPos[i] = maxBound - (newPos[i] - maxBound) // Reflect position
			t.velocity[i] *= -0.8                         // Reverse and dampen velocity component
		}
	}

	t.position = newPos // Update the position
}

// String representation for logging
func (t *Target) String() string {
	return fmt.Sprintf("Target[%s] Pos: %s Vel: %s", t.id, t.position, t.velocity)
}

// Initialize random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}
