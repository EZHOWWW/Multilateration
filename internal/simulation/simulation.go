package simulation

import (
	"fmt"
	"math/rand"
	"multilateration-sim/internal/common"          // Замените на ваше имя модуля
	"multilateration-sim/internal/multilateration" // Импортируем наш солвер
	"strings"
	"time"
)

// Simulation holds the state of the n-dimensional simulation.
type Simulation struct {
	dimension      int
	bounds         []float64                   // Simulation space boundaries [minX, maxX, minY, maxY, ...]
	objects        map[string]SimulationObject // All objects in the simulation, mapped by ID
	sensors        map[string]*Sensor          // Quick access to sensors
	targets        map[string]*Target          // Quick access to targets
	simulationTime float64                     // Total elapsed simulation time
	tickDuration   time.Duration               // How much real time corresponds to one simulation step (Update call)

	// Store last known estimated positions and errors for visualization later
	lastEstimates map[string]multilateration.Solution // Map target ID to last Solution
	lastErrors    map[string]float64                  // Map target ID to last localization error distance
}

// NewSimulation creates a new simulation environment.
func NewSimulation(dimension int, bounds []float64, tickDuration time.Duration) (*Simulation, error) {
	if len(bounds) != dimension*2 {
		return nil, fmt.Errorf("bounds length must be dimension * 2, got %d, expected %d", len(bounds), dimension*2)
	}
	if dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive, got %d", dimension)
	}

	return &Simulation{
		dimension:      dimension,
		bounds:         bounds,
		objects:        make(map[string]SimulationObject),
		sensors:        make(map[string]*Sensor),
		targets:        make(map[string]*Target),
		simulationTime: 0.0,
		tickDuration:   tickDuration,
		lastEstimates:  make(map[string]multilateration.Solution),
		lastErrors:     make(map[string]float64),
	}, nil
}

// AddObject adds a simulation object to the simulation.
func (s *Simulation) AddObject(obj SimulationObject) error {
	if obj.GetPosition().Dimension() != s.dimension {
		return fmt.Errorf("object dimension %d does not match simulation dimension %d", obj.GetPosition().Dimension(), s.dimension)
	}
	id := obj.GetID()
	if _, exists := s.objects[id]; exists {
		return fmt.Errorf("object with ID %s already exists", id)
	}
	s.objects[id] = obj

	// Add to specific maps for easier access
	switch v := obj.(type) {
	case *Sensor:
		s.sensors[id] = v
	case *Target:
		s.targets[id] = v
		// Initialize estimate/error map for new target
		s.lastEstimates[id] = multilateration.Solution{Position: nil, ResidualError: -1} // Indicate no estimate yet
		s.lastErrors[id] = -1.0
	}
	return nil
}

// AddRandomSensor adds a sensor at a random position within bounds.
func (s *Simulation) AddRandomSensor(radius float64, noise NoiseFunction) error {
	pos, err := common.NewRandomVector(s.dimension, s.bounds)
	if err != nil {
		return fmt.Errorf("failed to generate random position for sensor: %w", err)
	}
	sensor := NewSensor(pos, radius, noise)
	return s.AddObject(sensor)
}

// AddRandomTarget adds a target at a random position within bounds.
func (s *Simulation) AddRandomTarget() error {
	pos, err := common.NewRandomVector(s.dimension, s.bounds)
	if err != nil {
		return fmt.Errorf("failed to generate random position for target: %w", err)
	}
	target := NewTarget(pos)
	return s.AddObject(target)
}

// GetObject returns an object by its ID.
func (s *Simulation) GetObject(id string) (SimulationObject, bool) {
	obj, exists := s.objects[id]
	return obj, exists
}

// GetSensors returns a slice of all sensors.
func (s *Simulation) GetSensors() []*Sensor {
	sensors := make([]*Sensor, 0, len(s.sensors))
	for _, sen := range s.sensors {
		sensors = append(sensors, sen)
	}
	return sensors
}

// GetTargets returns a slice of all targets.
func (s *Simulation) GetTargets() []*Target {
	targets := make([]*Target, 0, len(s.targets))
	for _, tar := range s.targets {
		targets = append(targets, tar)
	}
	return targets
}

// GetLastEstimate returns the last calculated position estimate and residual for a target.
func (s *Simulation) GetLastEstimate(targetID string) (multilateration.Solution, bool) {
	sol, ok := s.lastEstimates[targetID]
	return sol, ok
}

// GetLastLocalizationError returns the last calculated localization error distance for a target.
func (s *Simulation) GetLastLocalizationError(targetID string) (float64, bool) {
	errVal, ok := s.lastErrors[targetID]
	return errVal, ok
}

// Run executes the simulation loop for a given number of steps or until stopped.
func (s *Simulation) Run(numSteps int) {
	fmt.Printf("Starting simulation: Dimension=%d, Bounds=%v, TickDuration=%s\n", s.dimension, s.bounds, s.tickDuration)
	fmt.Println("Initial State:")
	s.PrintState()

	deltaTime := s.tickDuration.Seconds() // Time elapsed in each step

	for i := 0; i < numSteps; i++ {
		s.simulationTime += deltaTime
		fmt.Printf("\n--- Simulation Step %d (Time: %.2fs) ---\n", i+1, s.simulationTime)

		// 1. Update all objects (move targets, etc.)
		for _, obj := range s.objects {
			obj.Update(deltaTime, s.bounds)
		}

		// --- Logging Updated Positions ---
		fmt.Println("  Updated Positions:")
		for _, sen := range s.sensors {
			fmt.Printf("    %s\n", sen)
		}
		for _, tar := range s.targets {
			fmt.Printf("    %s\n", tar)
		}
		fmt.Println("  ---")

		// 2. Measurement Phase & Multilateration Phase
		fmt.Println("  Localization Attempts:")
		for _, tar := range s.targets {
			targetID := tar.GetID()
			targetMeasurements := make([]multilateration.Measurement, 0, len(s.sensors))
			measurementDetails := []string{} // For logging

			// Collect measurements from all sensors for this target
			for _, sen := range s.sensors {
				dist, inRange, err := sen.MeasureDistance(tar)
				if err != nil {
					// Log error but continue, maybe other sensors work
					fmt.Printf("    [%s] Error measuring from %s: %v\n", targetID, sen.GetID(), err)
					continue
				}
				if inRange {
					targetMeasurements = append(targetMeasurements, multilateration.Measurement{
						SensorPosition: sen.GetPosition(),
						Distance:       dist,
					})
					trueDist, _ := sen.GetPosition().Distance(tar.GetPosition())
					measurementDetails = append(measurementDetails, fmt.Sprintf("%s(d=%.2f|t=%.2f)", sen.GetID(), dist, trueDist))
				}
			}

			// Attempt localization if enough measurements are available
			requiredMeasurements := s.dimension + 1
			logPrefix := fmt.Sprintf("    Target %s (%d measurements [%s]):", targetID, len(targetMeasurements), strings.Join(measurementDetails, ", "))

			if len(targetMeasurements) >= requiredMeasurements {
				solution, err := multilateration.SolveLeastSquares(targetMeasurements, s.dimension)

				truePos := tar.GetPosition()
				if err != nil {
					fmt.Printf("%s Localization failed: %v\n", logPrefix, err)
					// Reset last estimate/error for this target
					s.lastEstimates[targetID] = multilateration.Solution{Position: nil, ResidualError: -1}
					s.lastErrors[targetID] = -1.0
				} else {
					estimatedPos := solution.Position
					residualErr := solution.ResidualError
					localizationErr, distErr := multilateration.CalculateLocalizationError(truePos, estimatedPos)

					errorStr := "N/A"
					if distErr == nil {
						errorStr = fmt.Sprintf("%.3f", localizationErr)
						s.lastErrors[targetID] = localizationErr // Store last error distance
					} else {
						s.lastErrors[targetID] = -1.0 // Indicate error calculation failed
					}

					// Store last estimate
					s.lastEstimates[targetID] = solution

					fmt.Printf("%s True Pos: %s -> Est Pos: %s (Error: %s, Residual: %.3f)\n",
						logPrefix, truePos, estimatedPos, errorStr, residualErr)
				}
			} else {
				fmt.Printf("%s Insufficient measurements (%d/%d) for localization.\n",
					logPrefix, len(targetMeasurements), requiredMeasurements)
				// Reset last estimate/error if not enough measurements
				s.lastEstimates[targetID] = multilateration.Solution{Position: nil, ResidualError: -1}
				s.lastErrors[targetID] = -1.0
			}
		}

		// Optional delay for console viewing
		// time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("\n--- Simulation Finished ---")
	s.PrintState()
}

// PrintState prints the current positions of all objects.
func (s *Simulation) PrintState() {
	fmt.Println("--- Current Simulation State ---")
	fmt.Printf("Time: %.2fs\n", s.simulationTime)
	fmt.Println("Sensors:")
	if len(s.sensors) == 0 {
		fmt.Println("  None")
	}
	for _, sen := range s.sensors {
		fmt.Printf("  %s\n", sen) // Uses String() method
	}
	fmt.Println("Targets:")
	if len(s.targets) == 0 {
		fmt.Println("  None")
	}
	for _, tar := range s.targets {
		lastEst, okEst := s.GetLastEstimate(tar.GetID())
		lastErr, okErr := s.GetLastLocalizationError(tar.GetID())
		estimateStr := "None"
		if okEst && lastEst.Position != nil {
			errStr := "N/A"
			if okErr && lastErr >= 0 {
				errStr = fmt.Sprintf("%.3f", lastErr)
			}
			estimateStr = fmt.Sprintf("Est: %s (Err: %s, Resid: %.3f)", lastEst.Position, errStr, lastEst.ResidualError)
		}
		fmt.Printf("  %s | %s\n", tar, estimateStr) // Uses String() method
	}
	fmt.Println("-----------------------------")
}

// Initialize random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}
