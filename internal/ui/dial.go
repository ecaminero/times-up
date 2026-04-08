package ui

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// Dial is a custom Fyne widget that renders a circular countdown timer.
type Dial struct {
	widget.BaseWidget
	remaining int
	duration  int
	isBreak   bool
}

// NewDial creates a Dial with the initial total duration in seconds.
func NewDial(duration int) *Dial {
	d := &Dial{remaining: duration, duration: duration}
	d.ExtendBaseWidget(d)
	return d
}

// Update refreshes the displayed values and triggers a repaint.
func (d *Dial) Update(remaining, duration int, isBreak bool) {
	d.remaining = remaining
	d.duration = duration
	d.isBreak = isBreak
	d.Refresh()
}

func (d *Dial) CreateRenderer() fyne.WidgetRenderer {
	r := &dialRenderer{dial: d}
	r.build()
	return r
}

// ---------------------------------------------------------------------------

type dialRenderer struct {
	dial    *Dial
	objects []fyne.CanvasObject
	size    fyne.Size // cached from last Layout call, reused by Refresh

	background *canvas.Circle
	arcLines   []*canvas.Line  // progress arc drawn as many short segments
	trackLines []*canvas.Line  // full-circle dim track behind the arc
	tickMarks  []*canvas.Line  // 60 minor/major tick marks
	needle     *canvas.Line
	centerDot  *canvas.Circle
	tipDot     *canvas.Circle // white dot at the needle tip
}

const (
	numArcSegments = 120 // arc resolution (higher = smoother)
	numTicks       = 60  // one tick per minute of the hour
)

func (r *dialRenderer) build() {
	r.background = &canvas.Circle{FillColor: color.NRGBA{R: 28, G: 28, B: 35, A: 255}}

	r.trackLines = make([]*canvas.Line, numArcSegments)
	for i := range r.trackLines {
		r.trackLines[i] = &canvas.Line{
			StrokeWidth: 4,
			StrokeColor: color.NRGBA{R: 55, G: 55, B: 68, A: 200},
		}
	}

	r.arcLines = make([]*canvas.Line, numArcSegments)
	for i := range r.arcLines {
		r.arcLines[i] = &canvas.Line{StrokeWidth: 7}
	}

	r.tickMarks = make([]*canvas.Line, numTicks)
	for i := range r.tickMarks {
		r.tickMarks[i] = &canvas.Line{}
	}

	r.needle = &canvas.Line{
		StrokeWidth:  3,
		StrokeColor:  color.NRGBA{R: 230, G: 230, B: 230, A: 255},
	}
	r.centerDot = &canvas.Circle{FillColor: color.NRGBA{R: 230, G: 230, B: 230, A: 255}}
	r.tipDot = &canvas.Circle{FillColor: color.NRGBA{R: 230, G: 230, B: 230, A: 255}}

	r.objects = nil
	r.objects = append(r.objects, r.background)
	for _, l := range r.trackLines {
		r.objects = append(r.objects, l)
	}
	for _, l := range r.tickMarks {
		r.objects = append(r.objects, l)
	}
	for _, l := range r.arcLines {
		r.objects = append(r.objects, l)
	}
	r.objects = append(r.objects, r.needle, r.centerDot, r.tipDot)
}

func (r *dialRenderer) Layout(size fyne.Size) {
	r.size = size
	cx := size.Width / 2
	cy := size.Height / 2
	radius := min32(cx, cy) - 10

	r.background.Move(fyne.NewPos(cx-radius, cy-radius))
	r.background.Resize(fyne.NewSize(radius*2, radius*2))

	r.layoutTicks(cx, cy, radius)
	r.layoutTrack(cx, cy, radius)
	r.updateArc(cx, cy, radius)
	r.updateNeedle(cx, cy, radius)

	dotR := float32(5)
	r.centerDot.Move(fyne.NewPos(cx-dotR, cy-dotR))
	r.centerDot.Resize(fyne.NewSize(dotR*2, dotR*2))
}

// Refresh is called whenever Update() triggers d.Refresh().
// We recompute the dynamic parts (arc, needle) using the stored size.
func (r *dialRenderer) Refresh() {
	if r.dial == nil || r.size.Width == 0 {
		return
	}
	cx := r.size.Width / 2
	cy := r.size.Height / 2
	radius := min32(cx, cy) - 10
	r.updateArc(cx, cy, radius)
	r.updateNeedle(cx, cy, radius)
	canvas.Refresh(r.dial)
}

// layoutTicks positions the 60 tick marks (major every 5 min).
func (r *dialRenderer) layoutTicks(cx, cy, radius float32) {
	innerMinor := radius * 0.83
	innerMajor := radius * 0.76
	outer := radius * 0.94

	for i := 0; i < numTicks; i++ {
		angle := tickAngle(i)
		isMajor := i%5 == 0
		inner := innerMinor
		if isMajor {
			inner = innerMajor
			r.tickMarks[i].StrokeWidth = 2
			r.tickMarks[i].StrokeColor = color.NRGBA{R: 210, G: 210, B: 210, A: 210}
		} else {
			r.tickMarks[i].StrokeWidth = 1
			r.tickMarks[i].StrokeColor = color.NRGBA{R: 140, G: 140, B: 140, A: 130}
		}
		x1 := cx + inner*float32(math.Cos(angle))
		y1 := cy + inner*float32(math.Sin(angle))
		x2 := cx + outer*float32(math.Cos(angle))
		y2 := cy + outer*float32(math.Sin(angle))
		r.tickMarks[i].Position1 = fyne.NewPos(x1, y1)
		r.tickMarks[i].Position2 = fyne.NewPos(x2, y2)
	}
}

// layoutTrack positions the dim full-circle background ring.
func (r *dialRenderer) layoutTrack(cx, cy, radius float32) {
	trackRadius := radius * 0.87
	startAngle := -math.Pi / 2
	for i := 0; i < numArcSegments; i++ {
		a1 := startAngle + float64(i)/numArcSegments*2*math.Pi
		a2 := startAngle + float64(i+1)/numArcSegments*2*math.Pi
		r.trackLines[i].Position1 = fyne.NewPos(
			cx+trackRadius*float32(math.Cos(a1)),
			cy+trackRadius*float32(math.Sin(a1)),
		)
		r.trackLines[i].Position2 = fyne.NewPos(
			cx+trackRadius*float32(math.Cos(a2)),
			cy+trackRadius*float32(math.Sin(a2)),
		)
	}
}

// updateArc redraws the colored progress arc. Called from both Layout and Refresh.
func (r *dialRenderer) updateArc(cx, cy, radius float32) {
	var arcColor color.Color
	if r.dial.isBreak {
		arcColor = color.NRGBA{R: 55, G: 135, B: 255, A: 240}
	} else {
		arcColor = color.NRGBA{R: 55, G: 215, B: 95, A: 240}
	}

	arcRadius := radius * 0.87
	startAngle := -math.Pi / 2

	var progress float64
	if r.dial.duration > 0 {
		elapsed := r.dial.duration - r.dial.remaining
		progress = float64(elapsed) / float64(r.dial.duration)
	}
	totalAngle := progress * 2 * math.Pi

	for i := 0; i < numArcSegments; i++ {
		a1 := startAngle + float64(i)/numArcSegments*2*math.Pi
		a2 := startAngle + float64(i+1)/numArcSegments*2*math.Pi
		if a1 < startAngle+totalAngle {
			r.arcLines[i].StrokeColor = arcColor
			r.arcLines[i].StrokeWidth = 7
		} else {
			// Fully transparent when not in progress (track layer shows through)
			r.arcLines[i].StrokeColor = color.NRGBA{A: 0}
			r.arcLines[i].StrokeWidth = 0
		}
		r.arcLines[i].Position1 = fyne.NewPos(
			cx+arcRadius*float32(math.Cos(a1)),
			cy+arcRadius*float32(math.Sin(a1)),
		)
		r.arcLines[i].Position2 = fyne.NewPos(
			cx+arcRadius*float32(math.Cos(a2)),
			cy+arcRadius*float32(math.Sin(a2)),
		)
	}
}

// updateNeedle repositions the needle and its tip dot. Called from both Layout and Refresh.
func (r *dialRenderer) updateNeedle(cx, cy, radius float32) {
	var progress float64
	if r.dial.duration > 0 {
		progress = float64(r.dial.remaining) / float64(r.dial.duration)
	}
	// Needle counts down from 12 o'clock clockwise
	angle := -math.Pi/2 + (1-progress)*2*math.Pi
	needleLen := radius * 0.68
	tipX := cx + needleLen*float32(math.Cos(angle))
	tipY := cy + needleLen*float32(math.Sin(angle))

	r.needle.Position1 = fyne.NewPos(cx, cy)
	r.needle.Position2 = fyne.NewPos(tipX, tipY)

	tipR := float32(4)
	r.tipDot.Move(fyne.NewPos(tipX-tipR, tipY-tipR))
	r.tipDot.Resize(fyne.NewSize(tipR*2, tipR*2))
}

func (r *dialRenderer) MinSize() fyne.Size { return fyne.NewSize(270, 270) }
func (r *dialRenderer) Destroy()           {}
func (r *dialRenderer) Objects() []fyne.CanvasObject { return r.objects }

// tickAngle converts a tick index (0 = 12 o'clock) to radians, clockwise.
func tickAngle(i int) float64 {
	return -math.Pi/2 + float64(i)/numTicks*2*math.Pi
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
