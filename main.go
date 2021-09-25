package main

import (
	"flag"
	"log"
	"runtime"
)

var (
	gameWidth  = 20
	gameHeight = 20

	shuffleDir = Empty // random
)

func hasTerm() bool {
	switch runtime.GOOS {
	case "ios", "android", "js":
		return false

	default:
		return true
	}
}

func main() {
	term := false

	flag.IntVar(&gameWidth, "width", gameWidth, "screen width")
	flag.IntVar(&gameHeight, "height", gameHeight, "screen height")
	audio := flag.Bool("audio", true, "play audio effects")
	sdir := flag.String("shuffle", "random", "shuffle direction (left, right, random)")

	if hasTerm() {
		flag.BoolVar(&term, "term", term, "terminal UI vs. graphics UI")
	}

	flag.Parse()

	if gameWidth <= 0 || gameHeight <= 0 {
		log.Fatal("invalid width or height")
	}

	gameWidth += 2  // add border
	gameHeight += 2 // to simplify boundary checks

	switch *sdir {
	case "l", "left":
		shuffleDir = Left

	case "r", "right":
		shuffleDir = Right
	}

	// Initialize audio
	if *audio {
		audioInit()
	}

	if term {
		termGame()
	} else {
		gioGame()
	}
}
