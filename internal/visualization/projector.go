package visualization

import (
	"fmt"
	"multilateration-sim/internal/common"     // Замените на ваше имя модуля
	"multilateration-sim/internal/simulation" // Замените на ваше имя модуля

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// Projector is an interface for dimensionality reduction techniques.
type Projector interface {
	// Project takes a slice of simulation objects and returns their 2D projections,
	// along with a map linking original object IDs to their 2D positions.
	Project(objects []simulation.SimulationObject) (map[string]common.Vector, error)
}

// PCAProjector uses Principal Component Analysis to project n-dimensional data to 2D.
type PCAProjector struct {
	targetDimension int
}

// NewPCAProjector creates a new PCA projector targeting 2D.
func NewPCAProjector() *PCAProjector {
	return &PCAProjector{targetDimension: 2}
}

// Project performs PCA on the positions of the given simulation objects.
// It returns a map of objectID to its new 2D common.Vector position.
func (p *PCAProjector) Project(objects []simulation.SimulationObject) (map[string]common.Vector, error) {
	if len(objects) == 0 {
		return make(map[string]common.Vector), nil // No objects, return empty map
	}

	sourceDim := objects[0].GetPosition().Dimension()
	if sourceDim < p.targetDimension {
		// If source dimension is already 2D (or 1D), we can't reduce to 2D meaningfully via PCA this way.
		// Or, if it's 2D, we can just return the original coordinates.
		// For simplicity, if sourceDim < targetDim, let's return an error or handle as a special case.
		// For now, if source is 2D, we'll just "project" by returning the original 2D coords.
		if sourceDim == 2 && p.targetDimension == 2 {
			projectedPositions := make(map[string]common.Vector, len(objects))
			for _, obj := range objects {
				projectedPositions[obj.GetID()] = obj.GetPosition().Clone()
			}
			return projectedPositions, nil
		}
		// If sourceDim is 1D and target is 2D, we could pad with a zero y-coordinate.
		if sourceDim == 1 && p.targetDimension == 2 {
			projectedPositions := make(map[string]common.Vector, len(objects))
			for _, obj := range objects {
				originalPos := obj.GetPosition()
				projectedPos := common.NewVector(2)
				projectedPos[0] = originalPos[0]
				projectedPos[1] = 0 // Pad with zero for the second dimension
				projectedPositions[obj.GetID()] = projectedPos
			}
			return projectedPositions, nil
		}
		return nil, fmt.Errorf("source dimension (%d) is less than target dimension (%d), PCA not applicable in this setup", sourceDim, p.targetDimension)
	}

	numSamples := len(objects)
	data := make([]float64, numSamples*sourceDim)
	objectIDs := make([]string, numSamples) // To map results back

	for i, obj := range objects {
		pos := obj.GetPosition()
		objectIDs[i] = obj.GetID()
		for j := 0; j < sourceDim; j++ {
			data[i*sourceDim+j] = pos[j]
		}
	}

	// Create a Gonum matrix from the data.
	// The matrix should have samples as rows and features (dimensions) as columns.
	matrix := mat.NewDense(numSamples, sourceDim, data)

	// Perform PCA.
	var pc stat.PC
	ok := pc.PrincipalComponents(matrix, nil) // nil for weights means all samples weighted equally
	if !ok {
		return nil, fmt.Errorf("PCA computation failed")
	}

	// Check explained variance (optional, for debugging/info)
	// variances := pc.VarsTo(nil)
	// fmt.Printf("PCA Variances explained by each component: %v\n", variances)

	// Reduce the dimensionality to targetDimension (2D).
	// k is the number of principal components to keep.
	k := p.targetDimension
	if sourceDim < k { // Should have been caught earlier, but defensive check
		k = sourceDim
	}

	var reduced mat.Dense
	var vec mat.Dense
	// pc.Reduce(&reduced, k, matrix) // Reduce projects data onto the first k principal components
	pc.VectorsTo(&vec)
	reduced.Mul(matrix, vec.Slice(0, sourceDim, 0, k))

	// Store the projected 2D coordinates.
	projectedPositions := make(map[string]common.Vector, numSamples)
	for i := 0; i < numSamples; i++ {
		id := objectIDs[i]
		pos2D := common.NewVector(p.targetDimension)
		for j := 0; j < p.targetDimension; j++ {
			if j < reduced.RawMatrix().Cols { // Ensure we don't go out of bounds if k < targetDimension
				pos2D[j] = reduced.At(i, j)
			} else {
				pos2D[j] = 0 // Pad with zero if k was less than targetDimension (e.g. sourceDim was 1)
			}
		}
		projectedPositions[id] = pos2D
	}

	return projectedPositions, nil
}
