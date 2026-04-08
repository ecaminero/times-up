package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"times-up/internal/audio"
	"times-up/internal/timer"
	myui "times-up/internal/ui"
)

const (
	defaultFocusMin = 25
	defaultBreakMin = 5
	defaultVolume   = 0.8
	defaultSound    = "bell"
)

// soundKeyFromLabel maps Spanish UI labels to audio type keys.
var soundKeyFromLabel = map[string]string{
	"Campana":        "bell",
	"Pitido":         "beep",
	"Campanilla":     "chime",
	"Doble campana":  "doble",
}

func main() {
	a := app.New()
	w := a.NewWindow("Pomodoro")
	w.Resize(fyne.NewSize(520, 580))
	w.SetFixedSize(true)

	// --- audio ---
	player, err := audio.New(defaultVolume)
	if err != nil {
		log.Printf("audio init failed (no sound): %v", err)
	}
	soundType := defaultSound

	// --- timer ---
	t := timer.New(defaultFocusMin, defaultBreakMin)

	// --- large time display via RichText heading ---
	timeSeg := &widget.TextSegment{
		Text: formatTime(defaultFocusMin * 60),
		Style: widget.RichTextStyle{
			Alignment: fyne.TextAlignCenter,
			SizeName:  theme.SizeNameHeadingText,
			TextStyle: fyne.TextStyle{Bold: true, Monospace: true},
		},
	}
	timeText := widget.NewRichText(timeSeg)
	timeText.Wrapping = fyne.TextWrapOff

	phaseLabel := widget.NewLabel("Foco")
	phaseLabel.Alignment = fyne.TextAlignCenter

	// --- dial ---
	dial := myui.NewDial(defaultFocusMin * 60)

	// --- icon buttons ---
	playBtn := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), nil)
	resetBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), nil)
	skipBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), nil)

	// --- settings labels ---
	focusValLabel := widget.NewLabel(fmt.Sprintf("Foco %d min", defaultFocusMin))
	focusValLabel.TextStyle = fyne.TextStyle{Bold: true}
	breakValLabel := widget.NewLabel(fmt.Sprintf("Descanso %d min", defaultBreakMin))
	breakValLabel.TextStyle = fyne.TextStyle{Bold: true}
	volumeValLabel := widget.NewLabel(fmt.Sprintf("Volumen %d%%", int(defaultVolume*100)))
	volumeValLabel.TextStyle = fyne.TextStyle{Bold: true}

	// --- settings sliders ---
	focusSlider := widget.NewSlider(1, 60)
	focusSlider.Value = defaultFocusMin
	focusSlider.Step = 1

	breakSlider := widget.NewSlider(1, 30)
	breakSlider.Value = defaultBreakMin
	breakSlider.Step = 1

	volumeSlider := widget.NewSlider(0, 100)
	volumeSlider.Value = defaultVolume * 100
	volumeSlider.Step = 1

	soundSelect := widget.NewSelect([]string{"Campana", "Pitido", "Campanilla", "Doble campana"}, nil)
	soundSelect.SetSelected("Campana")

	// --- helper: refresh time display ---
	setTime := func(remaining int) {
		timeSeg.Text = formatTime(remaining)
		timeText.Refresh()
	}

	// --- wire timer callback ---
	prevRemaining := defaultFocusMin * 60
	t.OnTick = func(remaining int, isBreak bool) {
		// A jump upward in remaining means the phase just rolled over → play sound.
		if remaining > prevRemaining {
			if player != nil {
				player.Play(soundType)
			}
		}
		prevRemaining = remaining

		_, duration, _, _ := t.State()
		dial.Update(remaining, duration, isBreak)
		setTime(remaining)
		if isBreak {
			phaseLabel.SetText("Descanso")
		} else {
			phaseLabel.SetText("Foco")
		}
	}

	// --- button actions ---
	playBtn.OnTapped = func() {
		_, _, running, _ := t.State()
		if running {
			t.Pause()
			playBtn.Icon = theme.MediaPlayIcon()
			playBtn.Refresh()
		} else {
			t.Start()
			playBtn.Icon = theme.MediaPauseIcon()
			playBtn.Refresh()
		}
	}

	resetBtn.OnTapped = func() {
		t.Reset()
		playBtn.Icon = theme.MediaPlayIcon()
		playBtn.Refresh()
		rem, dur, _, isBreak := t.State()
		dial.Update(rem, dur, isBreak)
		setTime(rem)
		if isBreak {
			phaseLabel.SetText("Descanso")
		} else {
			phaseLabel.SetText("Foco")
		}
	}

	skipBtn.OnTapped = func() {
		t.Skip()
		rem, dur, _, isBreak := t.State()
		dial.Update(rem, dur, isBreak)
		setTime(rem)
		if isBreak {
			phaseLabel.SetText("Descanso")
		} else {
			phaseLabel.SetText("Foco")
		}
	}

	// --- settings callbacks ---
	focusSlider.OnChanged = func(v float64) {
		mins := int(v)
		focusValLabel.SetText(fmt.Sprintf("Foco %d min", mins))
		_, _, running, _ := t.State()
		if !running {
			t.SetDurations(mins, int(breakSlider.Value))
			rem, dur, _, isBreak := t.State()
			dial.Update(rem, dur, isBreak)
			setTime(rem)
			_ = dur
		}
	}

	breakSlider.OnChanged = func(v float64) {
		mins := int(v)
		breakValLabel.SetText(fmt.Sprintf("Descanso %d min", mins))
		_, _, running, _ := t.State()
		if !running {
			t.SetDurations(int(focusSlider.Value), mins)
			rem, dur, _, isBreak := t.State()
			dial.Update(rem, dur, isBreak)
			setTime(rem)
			_ = dur
		}
	}

	volumeSlider.OnChanged = func(v float64) {
		volumeValLabel.SetText(fmt.Sprintf("Volumen %d%%", int(v)))
		if player != nil {
			player.SetVolume(v / 100)
		}
	}

	soundSelect.OnChanged = func(s string) {
		if key, ok := soundKeyFromLabel[s]; ok {
			soundType = key
		}
	}

	// Preview button plays the currently selected sound at the current volume.
	previewBtn := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		if player != nil {
			player.Play(soundType)
		}
	})

	// --- layout ---
	buttons := container.NewCenter(
		container.NewHBox(resetBtn, playBtn, skipBtn),
	)

	// Two-column settings: [Focus | Break] and [Volume | Sound]
	settings := container.NewVBox(
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			container.NewVBox(focusValLabel, focusSlider),
			container.NewVBox(breakValLabel, breakSlider),
		),
		container.NewGridWithColumns(2,
			container.NewVBox(volumeValLabel, volumeSlider),
			container.NewVBox(
				widget.NewLabel("Sonido"),
				container.NewBorder(nil, nil, nil, previewBtn, soundSelect),
			),
		),
	)

	content := container.NewVBox(
		container.NewCenter(timeText),
		container.NewCenter(phaseLabel),
		container.NewCenter(dial),
		buttons,
		settings,
	)

	w.SetContent(content)
	w.ShowAndRun()
}

func formatTime(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
