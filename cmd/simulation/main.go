package main

import (
	"fmt"
	"log"
	"math/rand"
	"multilateration-sim/internal/simulation" // Замените на ваше имя модуля
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano()) // Initialize random seed globally

	// --- Simulation Parameters ---
	simDimension := 2                                    // Начнем с 2D для простоты
	simBounds := []float64{-100.0, 100.0, -100.0, 100.0} // Границы: X от -100 до 100, Y от -100 до 100
	simTickDuration := time.Second / 10                  // 10 шагов симуляции в секунду (0.1s per step)
	numSensors := 4
	numTargets := 2
	numSteps := 50 // Количество шагов симуляции для запуска

	// --- Create Simulation ---
	sim, err := simulation.NewSimulation(simDimension, simBounds, simTickDuration)
	if err != nil {
		log.Fatalf("Error creating simulation: %v", err)
	}

	// --- Add Sensors ---
	// Добавим сенсоры с разным шумом
	noiseFuncs := []simulation.NoiseFunction{
		simulation.NoNoise,               // Без шума
		simulation.GaussianNoise(1.5),    // Гауссовский шум (std dev 1.5)
		simulation.UniformNoise(2.0),     // Равномерный шум (+/- 2.0)
		simulation.PercentageNoise(0.05), // 5% шум от дистанции
	}

	for i := 0; i < numSensors; i++ {
		// Циклически выбираем функцию шума для разнообразия
		// noiseFunc := noiseFuncs[i%len(noiseFuncs)]
		noiseFunc := noiseFuncs[0]
		err := sim.AddRandomSensor(150.0, noiseFunc) // Радиус детекции 150
		if err != nil {
			log.Printf("Warning: could not add sensor %d: %v", i, err)
		}
	}

	// --- Add Targets ---
	for i := 0; i < numTargets; i++ {
		err := sim.AddRandomTarget()
		if err != nil {
			log.Printf("Warning: could not add target %d: %v", i, err)
		}
	}

	// --- Run Simulation ---
	sim.Run(numSteps)

	fmt.Println("\nApplication finished.")
	// --- UI Integration (Placeholder) ---
	// Later, instead of sim.Run(), we will initialize Ebiten
	// and pass the simulation state to the Ebiten game loop for drawing.
	/*
	   vis, err := visualization.NewRenderer(sim) // Create visualizer
	   if err != nil {
	       log.Fatalf("Failed to create visualizer: %v", err)
	   }
	   ebiten.SetWindowSize(800, 600)
	   ebiten.SetWindowTitle("N-Dimensional Multilateration Simulation (2D Projection)")
	   if err := ebiten.RunGame(vis); err != nil { // Run Ebiten game loop
	       log.Fatal(err)
	   }
	*/
}
