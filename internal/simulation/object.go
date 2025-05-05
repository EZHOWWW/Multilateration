package simulation

import "multilateration-sim/internal/common" // Используем имя модуля, которое вы указали в go mod init

// SimulationObject defines the interface for any object within the simulation.
type SimulationObject interface {
	// GetPosition returns the current position of the object.
	GetPosition() common.Vector
	// SetPosition sets the position of the object.
	SetPosition(pos common.Vector) error
	// Update updates the state of the object based on the elapsed time.
	Update(deltaTime float64, bounds []float64) // bounds define the simulation space limits
	// GetID returns the unique identifier of the object.
	GetID() string
}
