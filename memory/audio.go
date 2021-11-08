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

const (
	AudioFlip  = 0
	AudioReset = 1
)

var (
	//go:embed assets/flip.wav
	wavFlip []byte

	//go:embed assets/revflip.wav
	wavReset []byte

	audioBuffer *beep.Buffer
	audioLimits [2]int
)

func audioInit() {
	audioFlip, format, err := wav.Decode(bytes.NewBuffer(wavFlip))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer audioFlip.Close()

	audioReset, _, err := wav.Decode(bytes.NewBuffer(wavReset))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer audioReset.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	audioBuffer = beep.NewBuffer(format)

	audioBuffer.Append(audioFlip)
	audioLimits[0] = audioBuffer.Len() // 0 to audioLimits[0]

	audioBuffer.Append(audioReset)
	audioLimits[1] = audioBuffer.Len() // audioLimits[0] to audioLimits[1]
}

func audioPlay(aid int) {
	if audioBuffer == nil {
		return
	}

	var s beep.StreamSeeker

	switch aid {
	case AudioFlip:
		s = audioBuffer.Streamer(0, audioLimits[0])

	case AudioReset:
		s = audioBuffer.Streamer(audioLimits[0], audioLimits[1])
	}

	speaker.Play(s)
}
