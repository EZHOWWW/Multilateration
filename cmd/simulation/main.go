package main

import (
	"fmt"
	"log"
	"math/rand"
	"multilateration-sim/internal/simulation"    // Замените на ваше имя модуля
	"multilateration-sim/internal/visualization" // Импортируем пакет визуализации
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// createBounds helper function (from previous version)
func createBounds(dim int, bound float64) []float64 {
	bounds := make([]float64, 0, 2*dim)
	if dim <= 0 { // Handle invalid dimension
		return []float64{}
	}
	for i := 0; i < dim; i++ {
		bounds = append(bounds, -bound, bound)
	}
	return bounds
}

const (
	screenWidth  = 1024
	screenHeight = 768
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// --- Simulation Parameters ---
	simDimension := 2
	worldBound := 100.0 // Max coordinate value for random placement
	simBounds := createBounds(simDimension, worldBound)

	simTickDuration := time.Second / 30 // Simulation steps per second (e.g., 20 Hz)
	// Ebiten runs at 60 FPS by default for rendering. Simulation can step slower.

	sim, err := simulation.NewSimulation(simDimension, simBounds, simTickDuration)
	if err != nil {
		log.Fatalf("Error creating simulation: %v", err)
	}

	// --- Add Sensors ---
	numSensors := 6       // Increased for better coverage in 3D
	sensorRadius := 100.0 // Detection radius
	noiseFuncs := []simulation.NoiseFunction{
		nil, // No noise
		simulation.GaussianNoise(1.0),
		simulation.UniformNoise(2.0),
		simulation.PercentageNoise(0.03),
		simulation.GaussianNoise(0.5),
		simulation.UniformNoise(1.0),
	}
	for i := 0; i < numSensors; i++ {
		// noiseFunc := noiseFuncs[i%len(noiseFuncs)]
		noiseFunc := noiseFuncs[0]
		err := sim.AddRandomSensor(sensorRadius, noiseFunc)
		if err != nil {
			log.Printf("Warning: could not add sensor %d: %v", i, err)
		}
	}

	// --- Add Targets ---
	numTargets := 4 // Increased targets
	for i := 0; i < numTargets; i++ {
		err := sim.AddRandomTarget()
		if err != nil {
			log.Printf("Warning: could not add target %d: %v", i, err)
		}
	}

	// --- Initialize Projector & Renderer ---
	projector := visualization.NewPCAProjector()
	ebitenRenderer := visualization.NewRenderer(sim, projector)

	// --- Ebiten Game Loop Setup ---
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("N-Мерная Мультилатерационная Симуляция (PCA в 2D)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled) // Allow window resizing

	// --- Simulation Control (Separate Goroutine or Ticker) ---
	// We want the simulation to step at its own pace (simTickDuration),
	// while Ebiten renders at its own pace (typically 60 FPS).

	go func() { // Run simulation stepping in a separate goroutine
		ticker := time.NewTicker(simTickDuration)
		defer ticker.Stop()
		for range ticker.C {
			sim.Step(simTickDuration.Seconds())       // Step the simulation
			if int(sim.GetCurrentTime()*10)%10 == 0 { // roughly every second if tick is 0.1s
				fmt.Printf("\n--- Sim Time: %.2fs ---\n", sim.GetCurrentTime())
				sim.LogCurrentState()
			}
		}
	}()

	// --- Start Ebiten Game Loop ---
	// The renderer's Update method will handle PCA projection based on the latest sim state.
	// The renderer's Draw method will draw it.
	fmt.Println("Запуск Ebiten UI...")
	if err := ebiten.RunGame(ebitenRenderer); err != nil {
		log.Fatalf("Ebiten RunGame error: %v", err)
	}

	fmt.Println("\nСимуляция завершена.")
}
