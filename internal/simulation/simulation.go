package simulation

import (
	"fmt"
	"math/rand"
	"multilateration-sim/internal/common" // Замените на ваше имя модуля
	"multilateration-sim/internal/multilateration"
	"strings"
	"time"
)

// Simulation holds the state of the n-dimensional simulation.
type Simulation struct {
	dimension      int
	bounds         []float64
	objects        map[string]SimulationObject
	sensors        map[string]*Sensor
	targets        map[string]*Target
	simulationTime float64
	tickDuration   time.Duration // Not directly used by Step, but kept for context

	lastEstimates map[string]multilateration.Solution
	lastErrors    map[string]float64
}

// NewSimulation creates a new simulation environment.
func NewSimulation(dimension int, bounds []float64, tickDuration time.Duration) (*Simulation, error) {
	if len(bounds) != dimension*2 && dimension > 0 { // Allow empty bounds for 0-dim (though unlikely)
		return nil, fmt.Errorf("bounds length must be dimension * 2, got %d, expected %d for dim %d", len(bounds), dimension*2, dimension)
	}
	if dimension < 0 { // Allow 0 dimension if it makes sense for some edge case, but typically >= 1
		return nil, fmt.Errorf("dimension must be non-negative, got %d", dimension)
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

// AddObject, AddRandomSensor, AddRandomTarget, GetObject, GetSensors, GetTargets,
// GetLastEstimate, GetLastLocalizationError - остаются без изменений с предыдущей версии.

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

	switch v := obj.(type) {
	case *Sensor:
		s.sensors[id] = v
	case *Target:
		s.targets[id] = v
		s.lastEstimates[id] = multilateration.Solution{Position: nil, ResidualError: -1}
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
	sensor := NewSensor(pos, radius, noise) // NewSensor handles nil noise
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

// GetAllObjects returns a slice of all simulation objects.
func (s *Simulation) GetAllObjects() []SimulationObject {
	all := make([]SimulationObject, 0, len(s.objects))
	for _, obj := range s.objects {
		all = append(all, obj)
	}
	return all
}

// GetCurrentTime returns the current simulation time.
func (s *Simulation) GetCurrentTime() float64 {
	return s.simulationTime
}

// Step performs one step of the simulation: updates objects and attempts localization.
func (s *Simulation) Step(deltaTime float64) {
	s.simulationTime += deltaTime

	// 1. Update all objects (move targets, etc.)
	for _, obj := range s.objects {
		obj.Update(deltaTime, s.bounds)
	}

	// 2. Measurement Phase & Multilateration Phase (for each target)
	for _, tar := range s.targets {
		targetID := tar.GetID()
		targetMeasurements := make([]multilateration.Measurement, 0, len(s.sensors))

		for _, sen := range s.sensors {
			dist, inRange, err := sen.MeasureDistance(tar)
			if err != nil {
				// Log error internally or decide how to handle; for now, skip this measurement
				fmt.Printf("    [Internal Log - Target %s] Error measuring from %s: %v\n", targetID, sen.GetID(), err)
				continue
			}
			if inRange {
				targetMeasurements = append(targetMeasurements, multilateration.Measurement{
					SensorPosition: sen.GetPosition(),
					Distance:       dist,
				})
			}
		}

		requiredMeasurements := s.dimension + 1
		if len(targetMeasurements) >= requiredMeasurements {
			solution, err := multilateration.SolveLeastSquares(targetMeasurements, s.dimension)
			if err == nil {
				s.lastEstimates[targetID] = solution
				truePos := tar.GetPosition()
				localizationErr, distErr := multilateration.CalculateLocalizationError(truePos, solution.Position)
				if distErr == nil {
					s.lastErrors[targetID] = localizationErr
				} else {
					s.lastErrors[targetID] = -1.0 // Error calculating error
				}
			} else {
				// Localization failed
				s.lastEstimates[targetID] = multilateration.Solution{Position: nil, ResidualError: -1}
				s.lastErrors[targetID] = -1.0
				// fmt.Printf("    [Internal Log - Target %s] Localization failed: %v\n", targetID, err)
			}
		} else {
			// Insufficient measurements
			s.lastEstimates[targetID] = multilateration.Solution{Position: nil, ResidualError: -1}
			s.lastErrors[targetID] = -1.0
		}
	}
}

// LogCurrentState prints the current state of object positions and localization attempts.
func (s *Simulation) LogCurrentState() {
	fmt.Println("  Updated Positions:")
	for _, sen := range s.sensors { // Log sensors first
		fmt.Printf("    %s\n", sen)
	}
	for _, tar := range s.targets { // Then targets
		fmt.Printf("    %s\n", tar)
	}
	fmt.Println("  ---")
	fmt.Println("  Localization Results:")
	for _, tar := range s.targets {
		targetID := tar.GetID()
		truePos := tar.GetPosition()
		solution, estOk := s.lastEstimates[targetID]
		locErr, errOk := s.lastErrors[targetID]

		// Reconstruct measurement details for logging (optional, can be verbose)
		measurementDetails := []string{}
		numActualMeasurements := 0
		for _, sen := range s.sensors {
			dist, inRange, _ := sen.MeasureDistance(tar) // Ignoring error here for brevity
			if inRange {
				numActualMeasurements++
				trueDist, _ := sen.GetPosition().Distance(tar.GetPosition())
				measurementDetails = append(measurementDetails, fmt.Sprintf("%s(d=%.2f|t=%.2f)", sen.GetID(), dist, trueDist))
			}
		}
		logPrefix := fmt.Sprintf("    Target %s (%d measurements [%s]):", targetID, numActualMeasurements, strings.Join(measurementDetails, ", "))

		if estOk && solution.Position != nil {
			errorStr := "N/A"
			if errOk && locErr >= 0 {
				errorStr = fmt.Sprintf("%.3f", locErr)
			}
			fmt.Printf("%s True Pos: %s -> Est Pos: %s (Error: %s, Residual: %.3f)\n",
				logPrefix, truePos, solution.Position, errorStr, solution.ResidualError)
		} else {
			requiredMeasurements := s.dimension + 1
			if numActualMeasurements < requiredMeasurements {
				fmt.Printf("%s Insufficient measurements (%d/%d) for localization.\n",
					logPrefix, numActualMeasurements, requiredMeasurements)
			} else {
				fmt.Printf("%s Localization failed or no estimate available.\n", logPrefix)
			}
		}
	}
}

// PrintState prints the initial/final summary state of the simulation.
func (s *Simulation) PrintState() {
	fmt.Println("--- Simulation State Summary ---")
	fmt.Printf("Time: %.2fs, Dimension: %d\n", s.simulationTime, s.dimension)
	fmt.Println("Sensors:")
	if len(s.sensors) == 0 {
		fmt.Println("  None")
	}
	for _, sen := range s.sensors {
		fmt.Printf("  %s\n", sen)
	}
	fmt.Println("Targets:")
	if len(s.targets) == 0 {
		fmt.Println("  None")
	}
	for _, tar := range s.targets {
		lastEst, okEst := s.GetLastEstimate(tar.GetID())
		lastErr, okErr := s.GetLastLocalizationError(tar.GetID())
		estimateStr := "No estimate yet."
		if okEst && lastEst.Position != nil {
			errStr := "N/A"
			if okErr && lastErr >= 0 {
				errStr = fmt.Sprintf("%.3f", lastErr)
			}
			estimateStr = fmt.Sprintf("Last Est: %s (Err: %s, Resid: %.3f)", lastEst.Position, errStr, lastEst.ResidualError)
		}
		fmt.Printf("  %s | %s\n", tar, estimateStr)
	}
	fmt.Println("-----------------------------")
}

// Run (old version, kept for reference or if needed for non-Ebiten runs)
func (s *Simulation) RunLegacy(numSteps int) {
	fmt.Printf("Starting simulation: Dimension=%d, Bounds=%v, TickDuration=%s\n", s.dimension, s.bounds, s.tickDuration)
	fmt.Println("Initial State:")
	s.PrintState()

	deltaTime := s.tickDuration.Seconds()

	for i := 0; i < numSteps; i++ {
		fmt.Printf("\n--- Simulation Step %d (Time: %.2fs) ---\n", i+1, s.GetCurrentTime())
		s.Step(deltaTime)
		s.LogCurrentState()
		// time.Sleep(50 * time.Millisecond) // Optional delay
	}

	fmt.Println("\n--- Simulation Finished ---")
	s.PrintState()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (s *Simulation) GetDimension() int {
	return s.dimension
}
