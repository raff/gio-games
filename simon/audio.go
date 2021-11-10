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

const audioBuzz = 4

var (
	//go:embed assets/simonSound1.wav
	wav1 []byte

	//go:embed assets/simonSound2.wav
	wav2 []byte

	//go:embed assets/simonSound3.wav
	wav3 []byte

	//go:embed assets/simonSound4.wav
	wav4 []byte

	//go:embed assets/buzz.wav
	wavBuzz []byte

	audioBuffer *beep.Buffer
	audioLimits [5]int

	audioPlaying bool
)

func audioInit() {
	a1, format, err := wav.Decode(bytes.NewBuffer(wav1))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer a1.Close()

	a2, _, err := wav.Decode(bytes.NewBuffer(wav2))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer a2.Close()

	a3, _, err := wav.Decode(bytes.NewBuffer(wav3))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer a3.Close()

	a4, _, err := wav.Decode(bytes.NewBuffer(wav4))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer a4.Close()

	aBuzz, _, err := wav.Decode(bytes.NewBuffer(wavBuzz))
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer aBuzz.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	audioBuffer = beep.NewBuffer(format)

	audioBuffer.Append(a1)
	audioLimits[0] = audioBuffer.Len() // 0 to audioLimits[0]

	audioBuffer.Append(a2)
	audioLimits[1] = audioBuffer.Len() // audioLimits[0] to audioLimits[1]

	audioBuffer.Append(a3)
	audioLimits[2] = audioBuffer.Len() // audioLimits[1] to audioLimits[2]

	audioBuffer.Append(a4)
	audioLimits[3] = audioBuffer.Len() // audioLimits[2] to audioLimits[3]

	audioBuffer.Append(aBuzz)
	audioLimits[4] = audioBuffer.Len() // audioLimits[3] to audioLimits[4]
}

func audioPlay(aid int) {
	if audioBuffer == nil {
		return
	}

	if aid >= len(audioLimits) {
		return
	}

	var s beep.StreamSeeker

	if aid == 0 {
		s = audioBuffer.Streamer(0, audioLimits[0])
	} else {
		s = audioBuffer.Streamer(audioLimits[aid-1], audioLimits[aid])
	}

	audioPlaying = true

	speaker.Play(s, beep.Callback(func() {
		audioPlaying = false
	}))
}
