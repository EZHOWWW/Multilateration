package multilateration

import (
	"fmt"
	"math"
	"multilateration-sim/internal/common" // Замените на ваше имя модуля

	"gonum.org/v1/gonum/blas/blas64" // For vector norm calculation
	"gonum.org/v1/gonum/mat"         // Import the gonum matrix package
)

// Measurement represents a single distance measurement from a sensor.
type Measurement struct {
	SensorPosition common.Vector
	Distance       float64
}

// Solution contains the estimated position and a measure of the solution quality.
type Solution struct {
	Position      common.Vector
	ResidualError float64 // Lower is better. Represents ||Ax - b|| / sqrt(m)
}

// SolveLeastSquares attempts to find the target position using the least squares method.
// It requires at least dimension + 1 measurements for this linearized approach.
// Returns the estimated position and the normalized residual error.
func SolveLeastSquares(measurements []Measurement, dimension int) (Solution, error) {
	numMeasurements := len(measurements)
	var emptySolution Solution // Solution to return on error

	// We need at least n+1 measurements for n dimensions for the linearized system
	// to potentially have a unique solution via A^T A.
	if numMeasurements < dimension+1 {
		return emptySolution, fmt.Errorf("insufficient measurements: got %d, need at least %d for dimension %d for this LS method", numMeasurements, dimension+1, dimension)
	}

	// Use the last measurement's sensor as the reference sensor (k in the equations)
	refSensorPos := measurements[numMeasurements-1].SensorPosition
	refDist := measurements[numMeasurements-1].Distance
	if refDist < 0 {
		refDist = 0
	} // Ensure distance is non-negative
	refDistSq := refDist * refDist           // d_k^2
	refSensorNormSq := refSensorPos.NormSq() // ||S_k||^2 (Using our new method)

	// Create the matrix A (size (m-1) x n) and vector b (size (m-1) x 1)
	numEquations := numMeasurements - 1
	aData := make([]float64, numEquations*dimension)
	bData := make([]float64, numEquations)

	for i := 0; i < numEquations; i++ {
		sensorPos := measurements[i].SensorPosition // S_i
		dist := measurements[i].Distance
		if dist < 0 {
			dist = 0
		} // Ensure distance is non-negative
		distSq := dist * dist              // d_i^2
		sensorNormSq := sensorPos.NormSq() // ||S_i||^2 (Using our new method)

		// Calculate row i of matrix A: 2 * (S_k - S_i)
		diffVec, err := refSensorPos.Subtract(sensorPos)
		if err != nil {
			// This should not happen if dimensions are consistent
			return emptySolution, fmt.Errorf("dimension mismatch calculating A: %w", err)
		}
		scaledDiff := diffVec.MultiplyByScalar(2.0)
		for j := 0; j < dimension; j++ {
			aData[i*dimension+j] = scaledDiff[j]
		}

		// Calculate element i of vector b: d_i^2 - d_k^2 - ||S_i||^2 + ||S_k||^2
		bData[i] = distSq - refDistSq - sensorNormSq + refSensorNormSq
	}

	// Create gonum matrix objects
	A := mat.NewDense(numEquations, dimension, aData)
	b := mat.NewVecDense(numEquations, bData)

	// --- Solve the least squares problem A * x = b ---
	// We use QR decomposition directly as it's generally more robust for LS problems
	// than forming A^T A explicitly (which can worsen conditioning).
	var qr mat.QR
	qr.Factorize(A)

	// Check if the system might be rank-deficient (more likely with few sensors or poor geometry)
	// rank, _ := qr.Rank(1e-10) // Estimate rank with a tolerance
	rank := dimension
	if rank < dimension {
		fmt.Printf("Warning: System may be rank-deficient (rank %d < dimension %d). Solution might not be unique or reliable.\n", rank, dimension)
		// Continue solving, but the result's reliability is questionable.
	}

	var x mat.VecDense
	err := qr.SolveVecTo(&x, false, b) // Solves min ||Ax - b||_2
	if err != nil {
		// This might happen if A is severely ill-conditioned or has zero columns etc.
		return emptySolution, fmt.Errorf("QR least squares solve failed: %w", err)
	}

	// --- Calculate Residual Error ---
	var residualVec mat.VecDense
	residualVec.MulVec(A, &x)           // residualVec = A*x
	residualVec.SubVec(b, &residualVec) // residualVec = b - A*x
	// Use blas64 directly for norm calculation
	residualNorm := blas64.Nrm2(residualVec.RawVector())
	// Normalize the residual by sqrt(number of equations) for scale invariance
	normalizedResidual := residualNorm / math.Sqrt(float64(numEquations))

	// Extract the result into our common.Vector type
	resultVector := common.NewVector(dimension)
	for i := 0; i < dimension; i++ {
		resultVector[i] = x.AtVec(i)
	}

	solution := Solution{
		Position:      resultVector,
		ResidualError: normalizedResidual,
	}

	return solution, nil
}

// CalculateLocalizationError calculates the Euclidean distance between the true and estimated positions.
func CalculateLocalizationError(truePosition, estimatedPosition common.Vector) (float64, error) {
	if truePosition == nil || estimatedPosition == nil {
		return 0, fmt.Errorf("cannot calculate error with nil vectors")
	}
	// Ensure vectors are not nil before calculating distance
	if len(truePosition) == 0 || len(estimatedPosition) == 0 {
		return 0, fmt.Errorf("cannot calculate error with empty vectors")
	}
	return truePosition.Distance(estimatedPosition)
}
