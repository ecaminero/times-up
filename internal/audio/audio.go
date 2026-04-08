package audio

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/ebitengine/oto/v3"
)

const (
	sampleRate     = 44100
	channelCount   = 2
	bitDepth       = 16
	bytesPerSample = bitDepth / 8 * channelCount
)

// Player wraps an oto context and plays generated waveforms.
type Player struct {
	ctx    *oto.Context
	volume float64 // 0.0 – 1.0
}

// New initialises the audio output. Volume is clamped to [0,1].
func New(volume float64) (*Player, error) {
	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}

	opts := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channelCount,
		Format:       oto.FormatSignedInt16LE,
	}

	ctx, ready, err := oto.NewContext(opts)
	if err != nil {
		return nil, err
	}
	<-ready

	return &Player{ctx: ctx, volume: volume}, nil
}

// SetVolume updates the volume for subsequent Play calls.
func (p *Player) SetVolume(v float64) {
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	p.volume = v
}

// Play generates and plays the requested sound type.
// It is non-blocking; playback happens in a separate goroutine.
func (p *Player) Play(soundType string) {
	var pcm []byte
	switch soundType {
	case "chime":
		pcm = p.generateChime()
	case "beep":
		pcm = p.generateSquare(880, 0.18)
	case "doble":
		pcm = p.generateDoubleBell(440, 0.6)
	default: // "bell"
		pcm = p.generateSine(440, 0.45)
	}

	go func() {
		player := p.ctx.NewPlayer(bytes.NewReader(pcm))
		player.Play()
		// Wait for playback to finish before closing.
		for player.IsPlaying() {
		}
		player.Close()
	}()
}

// --- waveform generators ---

// generateSine produces a sine wave at freq Hz for durationSec seconds.
// An exponential decay envelope prevents clicks at the end.
func (p *Player) generateSine(freq, durationSec float64) []byte {
	n := int(sampleRate * durationSec)
	buf := make([]byte, n*bytesPerSample)
	for i := 0; i < n; i++ {
		t := float64(i) / sampleRate
		// Decay envelope: amplitude falls to ~0 at end
		env := math.Exp(-3.0 * t / durationSec)
		sample := math.Sin(2*math.Pi*freq*t) * env * p.volume
		s := int16(sample * math.MaxInt16)
		// Write the same sample to both L and R channels
		binary.LittleEndian.PutUint16(buf[i*bytesPerSample:], uint16(s))
		binary.LittleEndian.PutUint16(buf[i*bytesPerSample+2:], uint16(s))
	}
	return buf
}

// generateSquare produces a square wave (harsh beep) at freq Hz.
func (p *Player) generateSquare(freq, durationSec float64) []byte {
	n := int(sampleRate * durationSec)
	buf := make([]byte, n*bytesPerSample)
	period := sampleRate / freq
	for i := 0; i < n; i++ {
		t := float64(i) / sampleRate
		env := math.Exp(-5.0 * t / durationSec)
		var raw float64
		if math.Mod(float64(i), period) < period/2 {
			raw = 1.0
		} else {
			raw = -1.0
		}
		sample := raw * env * p.volume
		s := int16(sample * math.MaxInt16)
		binary.LittleEndian.PutUint16(buf[i*bytesPerSample:], uint16(s))
		binary.LittleEndian.PutUint16(buf[i*bytesPerSample+2:], uint16(s))
	}
	return buf
}

/*
	 generateDoubleBell mixes the fundamental (f) with its second harmonic (2f) at half the amplitude.
	 	The sum is normalized to 1.0 to avoid clipping.
		sample = ( sin(2π·f·t) + 0.5·sin(2π·2f·t) ) / 1.5 · envelope

		The harmonic adds brightness without losing the smooth body of the fundamental sine wave.
*/
func (p *Player) generateDoubleBell(freq, durationSec float64) []byte {
	n := int(sampleRate * durationSec)
	buf := make([]byte, n*bytesPerSample)
	for i := 0; i < n; i++ {
		t := float64(i) / sampleRate
		env := math.Exp(-2.5 * t / durationSec)

		raw := (math.Sin(2*math.Pi*freq*t) + 0.5*math.Sin(2*math.Pi*2*freq*t)) / 1.5
		sample := raw * env * p.volume
		s := int16(sample * math.MaxInt16)
		binary.LittleEndian.PutUint16(buf[i*bytesPerSample:], uint16(s))
		binary.LittleEndian.PutUint16(buf[i*bytesPerSample+2:], uint16(s))
	}
	return buf
}

// generateChime plays three descending sine tones in sequence (C5, G4, E4).
func (p *Player) generateChime() []byte {
	freqs := []float64{523.25, 392.0, 329.63} // C5, G4, E4
	var combined []byte
	for _, f := range freqs {
		combined = append(combined, p.generateSine(f, 0.3)...)
	}
	return combined
}
