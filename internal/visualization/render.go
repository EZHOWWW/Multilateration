package visualization

import (
	"fmt"
	"image/color"
	"math"
	"multilateration-sim/internal/common"     // Замените на ваше имя модуля
	"multilateration-sim/internal/simulation" // Замените на ваше имя модуля
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	objectRadiusOnScreen    = 5.0  // Базовый радиус объектов на экране
	predictedPosRadiusScale = 1.2  // Масштаб для круга предсказанной позиции
	padding                 = 50.0 // Отступ от краев экрана
)

var (
	sensorColorBase   = color.RGBA{0, 0, 255, 255} // Синий
	sensorRadiusColor = color.RGBA{0, 0, 200, 50}  // Полупрозрачный синий
	targetColorBase   = color.RGBA{255, 0, 0, 255} // Красный
	predictedPosColor = color.RGBA{255, 0, 0, 100} // Полупрозрачный красный
)

// Renderer implements ebiten.Game interface for visualization.
type Renderer struct {
	sim       *simulation.Simulation
	projector Projector // PCA projector

	screenWidth  int
	screenHeight int

	// Transformation parameters
	scale   float64
	offsetX float64
	offsetY float64

	// Cached projected coordinates
	projectedCoords map[string]common.Vector
}

// NewRenderer creates a new Ebiten renderer.
func NewRenderer(sim *simulation.Simulation, projector Projector) *Renderer {
	return &Renderer{
		sim:             sim,
		projector:       projector,
		projectedCoords: make(map[string]common.Vector),
		// screenWidth and screenHeight will be set by Layout
	}
}

// Update is called every tick.
// The simulation itself is stepped in the main game loop (main.go) before Ebiten's Update/Draw.
func (r *Renderer) Update() error {
	// Project all objects for the current frame
	allObjects := r.sim.GetAllObjects()
	if len(allObjects) > 0 {
		var err error
		r.projectedCoords, err = r.projector.Project(allObjects)
		if err != nil {
			// Log error, but don't stop the renderer; previous projection might still be usable or draw nothing
			fmt.Printf("Renderer Update: PCA Projection failed: %v\n", err)
			// Optionally, clear projectedCoords or handle error display
		}
	} else {
		r.projectedCoords = make(map[string]common.Vector) // Clear if no objects
	}

	// Recalculate transformation based on new projected coordinates
	r.calculateTransform()

	return nil
}

// calculateTransform determines the scaling and offset to fit projected points onto the screen.
func (r *Renderer) calculateTransform() {
	if len(r.projectedCoords) == 0 {
		r.scale = 1.0
		r.offsetX = float64(r.screenWidth) / 2.0
		r.offsetY = float64(r.screenHeight) / 2.0
		return
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, pos := range r.projectedCoords {
		if len(pos) >= 2 { // Ensure it's a 2D vector
			if pos[0] < minX {
				minX = pos[0]
			}
			if pos[0] > maxX {
				maxX = pos[0]
			}
			if pos[1] < minY {
				minY = pos[1]
			}
			if pos[1] > maxY {
				maxY = pos[1]
			}
		}
	}

	if minX == math.MaxFloat64 { // No valid points found
		r.scale = 1.0
		r.offsetX = float64(r.screenWidth) / 2.0
		r.offsetY = float64(r.screenHeight) / 2.0
		return
	}

	// If all points are the same, avoid division by zero
	worldWidth := maxX - minX
	worldHeight := maxY - minY

	if worldWidth == 0 && worldHeight == 0 { // Single point or all points identical
		r.scale = 1.0 // Or some default zoom
		r.offsetX = float64(r.screenWidth)/2.0 - minX*r.scale
		r.offsetY = float64(r.screenHeight)/2.0 - minY*r.scale
		return
	}

	if worldWidth == 0 {
		worldWidth = 1
	} // Avoid division by zero if points form a vertical line
	if worldHeight == 0 {
		worldHeight = 1
	} // Avoid division by zero if points form a horizontal line

	scaleX := (float64(r.screenWidth) - 2*padding) / worldWidth
	scaleY := (float64(r.screenHeight) - 2*padding) / worldHeight
	r.scale = math.Min(scaleX, scaleY) // Preserve aspect ratio

	// If scale is too small or NaN/Inf (e.g. worldWidth/Height was 0 initially and became 1)
	if r.scale <= 0 || math.IsNaN(r.scale) || math.IsInf(r.scale, 0) {
		r.scale = 1.0 // Default scale
	}

	// Center the world
	centerX := (minX + maxX) / 2.0
	centerY := (minY + maxY) / 2.0
	r.offsetX = float64(r.screenWidth)/2.0 - centerX*r.scale
	r.offsetY = float64(r.screenHeight)/2.0 - centerY*r.scale
}

// worldToScreen converts projected 2D world coordinates to screen coordinates.
func (r *Renderer) worldToScreen(worldX, worldY float64) (float32, float32) {
	screenX := worldX*r.scale + r.offsetX
	screenY := worldY*r.scale + r.offsetY // Ebiten Y is top-down. This mapping assumes PCA Y is also "up".
	return float32(screenX), float32(screenY)
}

// Draw is called every frame to render the simulation.
func (r *Renderer) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{230, 230, 230, 255}) // Light gray background

	if len(r.projectedCoords) == 0 && len(r.sim.GetAllObjects()) > 0 {
		ebitenutil.DebugPrint(screen, "Waiting for PCA projection...")
		return
	}

	// Draw Sensors and their detection radii
	for _, sensor := range r.sim.GetSensors() {
		projPos, ok := r.projectedCoords[sensor.GetID()]
		if !ok || len(projPos) < 2 {
			continue
		}
		sx, sy := r.worldToScreen(projPos[0], projPos[1])

		// Draw detection radius first (so sensor is on top)
		// Radius in world units needs to be scaled.
		// Note: PCA might distort circles. This draws a circle in the 2D projected space.
		detectionRadiusOnScreen := float32(sensor.DetectionRadius() * r.scale) // DetectionRadius() method needed in Sensor
		if detectionRadiusOnScreen > 0 {
			vector.DrawFilledCircle(screen, sx, sy, detectionRadiusOnScreen, sensorRadiusColor, true)
		}

		// Draw sensor
		vector.DrawFilledCircle(screen, sx, sy, float32(objectRadiusOnScreen), sensorColorBase, true)
	}

	// Draw Targets and their predicted positions
	for _, target := range r.sim.GetTargets() {
		targetID := target.GetID()
		projPos, ok := r.projectedCoords[targetID]
		if !ok || len(projPos) < 2 {
			continue
		}
		tx, ty := r.worldToScreen(projPos[0], projPos[1])

		// Draw predicted position (if available)
		lastEstimate, estOk := r.sim.GetLastEstimate(targetID)
		if estOk && lastEstimate.Position != nil {
			// We need to project the N-D estimated position to 2D as well.
			// This is tricky: PCA was done on true positions. Applying same transform might not be ideal.
			// For simplicity, we'll assume the error in N-D translates to a similar region in 2D.
			// A more robust way would be to include estimates in PCA or project separately.
			// For now, let's draw the predicted circle around the *projected true position*
			// if we don't have a direct 2D projection of the estimate.
			// OR, if the estimate is also N-D, we'd need to project it:
			// tempObjectsForPCA := []simulation.SimulationObject{simulation.NewPointObject("est", lastEstimate.Position)}
			// projectedEst, _ := r.projector.Project(tempObjectsForPCA)
			// if pest, pOk := projectedEst["est"]; pOk {
			//    esx, esy := r.worldToScreen(pest[0], pest[1])
			//    vector.DrawFilledCircle(screen, esx, esy, float32(objectRadiusOnScreen*predictedPosRadiusScale), predictedPosColor, true)
			// }
			// Simpler: just draw a circle around the target's projected true position as a placeholder for "estimated region"
			// This is not ideal but simpler for now.
			// Let's assume lastEstimate.Position is N-D. We need to project it.
			// This is a bit complex as PCA is fitted to ALL objects. Projecting one point might be unstable.
			// A simpler visual cue: draw the predicted circle near the true projected target.
			vector.DrawFilledCircle(screen, tx, ty, float32(objectRadiusOnScreen*predictedPosRadiusScale*2), predictedPosColor, true)
		}

		// Draw target as a triangle
		// vector.DrawFilledCircle(screen, tx, ty, float32(objectRadiusOnScreen), targetColorBase, true) // Alternative: circle
		path := &vector.Path{}
		path.MoveTo(tx, ty-float32(objectRadiusOnScreen*1.5))                                   // Top point
		path.LineTo(tx-float32(objectRadiusOnScreen*1.2), ty+float32(objectRadiusOnScreen*0.8)) // Bottom-left
		path.LineTo(tx+float32(objectRadiusOnScreen*1.2), ty+float32(objectRadiusOnScreen*0.8)) // Bottom-right
		path.Close()
		// vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
		// vector.DrawVertices(screen, vs, is, targetColorBase, &ebiten.DrawTrianglesOptions{AntiAlias: true})
		vector.DrawFilledCircle(screen, tx, ty, 5, targetColorBase, true)

	}

	// Draw Debug Info
	r.drawDebugInfo(screen)
}

func (r *Renderer) drawDebugInfo(screen *ebiten.Image) {
	simTime := r.sim.GetCurrentTime()
	msg := fmt.Sprintf("Время симуляции: %.2fs\n", simTime)
	msg += fmt.Sprintf("FPS: %.1f, TPS: %.1f\n", ebiten.ActualFPS(), ebiten.ActualTPS())
	msg += fmt.Sprintf("Размерность: %dD -> 2D (PCA)\n", r.sim.GetDimension()) // GetDimension() method needed

	var totalError float64
	var numErrors int
	for _, target := range r.sim.GetTargets() {
		errVal, ok := r.sim.GetLastLocalizationError(target.GetID())
		if ok && errVal >= 0 {
			totalError += errVal
			numErrors++
		}
	}
	avgError := 0.0
	if numErrors > 0 {
		avgError = totalError / float64(numErrors)
		msg += fmt.Sprintf("Средняя ошибка локализации: %.3f\n", avgError)
	} else {
		msg += "Средняя ошибка локализации: N/A\n"
	}

	// Display object counts
	msg += fmt.Sprintf("Сенсоры: %d, Цели: %d\n", len(r.sim.GetSensors()), len(r.sim.GetTargets()))

	// Display detailed info for each target
	targetInfoLines := []string{"Информация по целям:"}
	for _, target := range r.sim.GetTargets() {
		line := fmt.Sprintf("  %s: Истин. %s", target.GetID(), target.GetPosition())
		est, estOk := r.sim.GetLastEstimate(target.GetID())
		if estOk && est.Position != nil {
			line += fmt.Sprintf(" | Оценка %s (Res: %.2f)", est.Position, est.ResidualError)
		} else {
			line += " | Оценка: нет"
		}
		locErr, errOk := r.sim.GetLastLocalizationError(target.GetID())
		if errOk && locErr >= 0 {
			line += fmt.Sprintf(" (Err: %.2f)", locErr)
		}
		targetInfoLines = append(targetInfoLines, line)
	}
	msg += strings.Join(targetInfoLines, "\n")

	ebitenutil.DebugPrint(screen, msg)
}

// Layout is called when the window size changes.
func (r *Renderer) Layout(outsideWidth, outsideHeight int) (int, int) {
	r.screenWidth = outsideWidth
	r.screenHeight = outsideHeight
	// The transform (scale, offset) will be recalculated in Update/Draw based on new screen size
	return r.screenWidth, r.screenHeight
}

// Helper methods to be added to simulation.Sensor and simulation.Simulation:
// simulation.Sensor.DetectionRadius() float64
// simulation.Simulation.GetDimension() int
