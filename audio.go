//go:build !ios && !android && !js

package main

import (
	"bytes"
	"log"
	"time"

	_ "embed"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

var (
	//go:embed assets/remove.wav
	wavRemove []byte

	//go:embed assets/move.wav
	wavMove []byte

	//go:embed assets/stop.wav
	wavStop []byte

	//go:embed assets/shuffle.wav
	wavShuffle []byte

	audioBuffer *beep.Buffer
	audioLimits [5]int
)

func reverse(s beep.Streamer, n int) beep.Streamer {
	rev := make([][2]float64, n)
	if nn, ok := s.Stream(rev); nn != n || !ok {
		log.Fatalf("cannot stream for reverse: %v/%v %v", nn, n, ok)
	}

	pos := n - 1

	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		for i := 0; i < len(samples) && pos >= 0; i++ {
			samples[i] = rev[pos]
			pos--
			n++
		}

		return n, pos >= 0
	})
}

func audioInit() {
	audioRemove, format, err := wav.Decode(bytes.NewBuffer(wavRemove))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer audioRemove.Close()

	audioMove, _, err := wav.Decode(bytes.NewBuffer(wavMove))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer audioMove.Close()

	audioStop, _, err := wav.Decode(bytes.NewBuffer(wavStop))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer audioStop.Close()

	audioShuffle, _, err := wav.Decode(bytes.NewBuffer(wavShuffle))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer audioShuffle.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	audioBuffer = beep.NewBuffer(format)

	audioBuffer.Append(audioRemove)
	audioLimits[0] = audioBuffer.Len() // 0 to audioLimits[0]

	audioBuffer.Append(audioMove)
	audioLimits[1] = audioBuffer.Len() // audioLimits[0] to audioLimits[1]

	audioBuffer.Append(audioStop)
	audioLimits[2] = audioBuffer.Len() // audioLimits[1] to audioLimits[2]

	audioBuffer.Append(audioShuffle)
	audioLimits[3] = audioBuffer.Len() // audioLimits[2] to audioLimits[3]

	s := audioBuffer.Streamer(0, audioLimits[0])
	audioBuffer.Append(reverse(s, audioLimits[0]))
	audioLimits[4] = audioBuffer.Len() // audioLimits[3] to audioLimits[4]
}

func audioPlay(mov Updates) {
	if audioBuffer == nil || mov == Invalid {
		return
	}

	var s beep.StreamSeeker

	switch mov {
	case Remove:
		s = audioBuffer.Streamer(0, audioLimits[0])

	case Move:
		s = audioBuffer.Streamer(audioLimits[0], audioLimits[1])

	case None:
		s = audioBuffer.Streamer(audioLimits[1], audioLimits[2])

	case Shuffle:
		s = audioBuffer.Streamer(audioLimits[2], audioLimits[3])

	case Undo:
		s = audioBuffer.Streamer(audioLimits[3], audioLimits[4])
	}

	speaker.Play(s)
}
