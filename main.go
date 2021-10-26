package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

var (
	gameWidth  = 20
	gameHeight = 20

	shuffleDir = Empty // random
	scorefile  = os.ExpandEnv("${HOME}/.arrows")
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
	score := flag.Bool("score", false, "display scoreboard")

	if hasTerm() {
		flag.BoolVar(&term, "term", term, "terminal UI vs. graphics UI")
	}

	flag.Parse()

	if gameWidth <= 0 || gameHeight <= 0 {
		log.Fatal("invalid width or height")
	}

	if f, err := os.Open(scorefile); err == nil {
		dec := json.NewDecoder(f)
		if err := dec.Decode(&scores); err != nil {
			log.Printf("cannot read %v: %v", scorefile, err)
		}
		f.Close()
	}

	defer terminateMain()

	gameWidth += 2  // add border
	gameHeight += 2 // to simplify boundary checks

	if *score {
		fmt.Println()
		fmt.Println("       Scoreboard")
		fmt.Println("     Moves Seq Score")

		for i, s := range scores.Get(gameWidth, gameHeight) {
			fmt.Printf("%2d:  %4d  %3d %5d\n", i+1, s.Moves, s.MaxSeq, s.Score)
		}

		return
	}

	switch *sdir {
	case "l", "left":
		shuffleDir = Left

	case "r", "right":
		shuffleDir = Right

	default:
		shuffleDir = Empty
	}

	// Initialize audio
	if *audio {
		audioInit()
	}

	if term {
		termGame(terminateMain)
	} else {
		gioGame(terminateMain)
	}
}

func terminateMain() {
	if f, err := os.Create(scorefile); err == nil {
		enc := json.NewEncoder(f)
		if err := enc.Encode(scores); err != nil {
			log.Printf("cannot write %v: %v", scorefile, err)
		}
		f.Close()
	} else {
		log.Println(err)
	}

	os.Exit(0)
}
