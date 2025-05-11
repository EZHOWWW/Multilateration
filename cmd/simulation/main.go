package main

import (
	"fmt"
	"log"
	"math/rand"
	"multilateration-sim/internal/simulation"    // Замените на ваше имя модуля
	"multilateration-sim/internal/visualization" // Импортируем пакет визуализации
	"time"
)

// Create bounds: [-bound, bound] times dim
func createBounds(dim int, bound float64) []float64 {
	bounds := make([]float64, 0, 2*dim)
	for i := 0; i < dim; i++ {
		bounds = append(bounds, -bound, bound)
	}
	return bounds
}
func main() {
	rand.Seed(time.Now().UnixNano()) // Initialize random seed globally

	// --- Simulation Parameters ---
	simDimension := 3
	bound := 100.0
	simBounds := createBounds(simDimension, bound)
	// --- Create Simulation ---
	simTickDuration := time.Second / 10
	sim, err := simulation.NewSimulation(simDimension, simBounds, simTickDuration)
	if err != nil {
		log.Fatalf("Error creating simulation: %v", err)
	}

	// --- Add Sensors ---
	numSensors := 5
	sensorRadius := 180.0
	noiseFuncs := []simulation.NoiseFunction{
		nil,
		simulation.GaussianNoise(2.5),
		simulation.UniformNoise(3.0),
		simulation.PercentageNoise(0.08),
		simulation.GaussianNoise(1.0),
	}

	for i := 0; i < numSensors; i++ {
		noiseFunc := noiseFuncs[i%len(noiseFuncs)]
		err := sim.AddRandomSensor(sensorRadius, noiseFunc)
		if err != nil {
			log.Printf("Warning: could not add sensor %d: %v", i, err)
		}
	}

	// --- Add Targets ---
	numTargets := 3
	for i := 0; i < numTargets; i++ {
		err := sim.AddRandomTarget()
		if err != nil {
			log.Printf("Warning: could not add target %d: %v", i, err)
		}
	}

	// --- Initialize Projector ---
	projector := visualization.NewPCAProjector()

	// --- Run Simulation (демонстрация PCA в цикле) ---
	fmt.Printf("Starting simulation: Dimension=%d, Bounds=%v, TickDuration=%s\n", simDimension, simBounds, simTickDuration)
	sim.PrintState() // Print initial state before first step

	deltaTime := simTickDuration.Seconds()

	numSteps := 10 // Уменьшим количество шагов для краткости вывода
	for i := 0; i < numSteps; i++ {
		sim.Step(deltaTime) // Используем новый метод Sim.Step() (мы его добавим)

		fmt.Printf("\n--- Simulation Step %d (Time: %.2fs) ---\n", i+1, sim.GetCurrentTime()) // Метод для времени
		sim.LogCurrentState()                                                                 // Метод для логирования состояния

		// --- Perform Projection ---
		allObjects := sim.GetAllObjects() // Нам нужен метод для получения всех объектов
		projectedCoords, err := projector.Project(allObjects)
		if err != nil {
			log.Printf("Step %d: PCA Projection failed: %v", i+1, err)
		} else {
			fmt.Println("  Projected 2D Coordinates (PCA):")
			for id, pos2d := range projectedCoords {
				originalObj, _ := sim.GetObject(id)
				fmt.Printf("    Object %s (Original: %s) -> Projected: %s\n", id, originalObj.GetPosition(), pos2d)
			}
		}
	}

	fmt.Println("\n--- Simulation Finished ---")
	sim.PrintState() // Print final state

	fmt.Println("\nApplication finished.")
}

// Для работы этого main.go, нам нужно добавить несколько методов в Simulation:
// 1. Step(deltaTime float64) - выполняет один шаг обновления и локализации.
// 2. GetCurrentTime() float64 - возвращает s.simulationTime.
// 3. LogCurrentState() - печатает информацию о локализации и позициях (как в старом Run).
// 4. GetAllObjects() []SimulationObject - возвращает срез всех объектов.
// Это сделает основной цикл в main.go чище и подготовит Simulation к интеграции с Ebiten.
