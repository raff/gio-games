package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "embed"

	"github.com/raff/arrows/game"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/gdamore/tcell"
)

const (
	up    = '\u2b06' // '\u2191'
	down  = '\u2b07' // '\u2193'
	left  = '\u2b05' // '\u2190'
	right = '\u2b95' // '\u2192'
	empty = ' '

	cw = 2
	ch = 1
)

var (
	width  = 20
	height = 20

	sx = 2
	sy = 2

	dirs  = []rune{empty, up, down, left, right}
	agame game.Game

	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	revStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorGreen)

	//go:embed audio/remove.wav
	wavRemove []byte

	//go:embed audio/move.wav
	wavMove []byte

	//go:embed audio/stop.wav
	wavStop []byte

	//go:embed audio/shuffle.wav
	wavShuffle []byte

	audioBuffer *beep.Buffer
	audioLimits [5]int
)

func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func drawScreen(s tcell.Screen) {
	x1 := sx
	y1 := sy
	x2 := x1 + (agame.Width * 2) + 1
	y2 := y1 + agame.Height + 1
	style := boxStyle

	// Fill screen
	for y, row := range agame.Screen {
		for x, col := range row {
			s.SetContent(x1+(2*x)+1, y1+y+1, dirs[col], nil, style)
		}
	}

	// Draw borders
	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}

	// Only draw corners if necessary
	if y1 != y2 && x1 != x2 {
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}
}

func checkScreen(s tcell.Screen, x, y int, op game.Updates) (cx, cy int, mov game.Updates) {
	msg := "                                         "

	cx, cy, mov = agame.Update(x-sx-1, y-sy-1, op)
	if mov != game.Invalid {
		s.ShowCursor(agame.ScreenCoords(sx+1, sy+1, cx, cy))
		msg = fmt.Sprintf("moves=%v remain=%v removed=%v seq=%v       ", agame.Moves, agame.Count, agame.Removed, agame.Seq)
	}

	drawScreen(s)
	drawText(s, sx, sy+height+2, sx+len(msg)+1, sy+height+2, boxStyle, msg)

	return
}

func centerScreen(s tcell.Screen) (int, int, bool) {
	gw, gh := agame.Width*2+2, agame.Height+2
	w, h := s.Size()

	px, py := sx, sy

	if w > gw {
		sx = (w - gw) / 2 // center horizontally
	}

	if h > gh {
		sy = (h - gh) / 2 // center vertically
	}

	if sx != px || sy != py {
		s.Clear()
		return agame.Coords(px, py)
	}

	return -1, -1, false
}

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

func audioPlay(mov game.Updates) {
	if audioBuffer == nil || mov == game.Invalid {
		return
	}

	var s beep.StreamSeeker

	switch mov {
	case game.Remove:
		s = audioBuffer.Streamer(0, audioLimits[0])

	case game.Move:
		s = audioBuffer.Streamer(audioLimits[0], audioLimits[1])

	case game.None:
		s = audioBuffer.Streamer(audioLimits[1], audioLimits[2])

	case game.Shuffle:
		s = audioBuffer.Streamer(audioLimits[2], audioLimits[3])

	case game.Undo:
		s = audioBuffer.Streamer(audioLimits[3], audioLimits[4])
	}

	speaker.Play(s)
}

func main() {
	flag.IntVar(&width, "width", width, "screen width")
	flag.IntVar(&height, "height", height, "screen height")
	audio := flag.Bool("audio", true, "play audio effects")
	flag.Parse()

	if width <= 0 || height <= 0 {
		log.Fatal("invalid width or height")
	}

	width += 2  // add border
	height += 2 // to simplify boundary checks

	// Initialize audio
	if *audio {
		audioInit()
	}

	// Initialize screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.Clear()

	// Draw initial screen
	agame.Setup(width, height, cw, ch)
	drawScreen(s)

	// Event loop
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

	ops := map[bool]game.Updates{true: game.Move, false: game.None}

	cx, cy := agame.ScreenCoords(sx+1, sy+1, 1, 1)
	s.ShowCursor(cx, cy)

	for {
		// Update screen
		s.Show()

		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()

			if x, y, ok := centerScreen(s); ok {
				cx, cy = agame.ScreenCoords(sx+1, sy+1, x, y)
				s.ShowCursor(cx, cy)
				drawScreen(s)
			}

		case *tcell.EventKey:
			ckey, crune := ev.Key(), ev.Rune()

			if ckey == tcell.KeyEscape || ckey == tcell.KeyCtrlC {
				quit()
			} else if ckey == tcell.KeyCtrlL {
				s.Sync()
			} else if ckey == tcell.KeyUp {
				if _, _, mov := checkScreen(s, cx, cy-1, game.None); mov != game.Invalid {
					cy--
				}
			} else if ckey == tcell.KeyDown {
				if _, _, mov := checkScreen(s, cx, cy+1, game.None); mov != game.Invalid {
					cy++
				}
			} else if ckey == tcell.KeyLeft {
				if _, _, mov := checkScreen(s, cx-2, cy, game.None); mov != game.Invalid {
					cx -= 2
				}
			} else if ckey == tcell.KeyRight {
				if _, _, mov := checkScreen(s, cx+2, cy, game.None); mov != game.Invalid {
					cx += 2
				}
			} else if crune == ' ' { // hit
				_, _, mov := checkScreen(s, cx, cy, game.Move)
				audioPlay(mov)

				if agame.Count == 0 {
					if agame.Winner() {
						s.PostEvent(tcell.NewEventInterrupt(true))
					}
				}
			} else if crune == 'U' || crune == 'u' { // undo
				if x, y, ok := agame.Undo(); ok {
					audioPlay(game.Undo)
					cx, cy = agame.ScreenCoords(sx+1, sy+1, x, y)
					checkScreen(s, cx, cx, game.None)
				}
			} else if crune == 'R' || crune == 'r' { // reset
				audioPlay(game.Undo)
				agame.Setup(width, height, cw, ch)
				drawScreen(s)
			} else if crune == 'S' || crune == 's' { // reshuffle
				audioPlay(game.Shuffle)
				agame.Shuffle()
				drawScreen(s)
			} else if crune == 'H' || crune == 'h' { // remove all "free" arrows
				moved := game.Invalid

				for y := 1; y < agame.Height-1; y++ {
					for x := 1; x < agame.Width-1; x++ {
						x, y := agame.ScreenCoords(0, 0, x, y)
						_, _, mov := agame.Update(x, y, game.Remove)
						if mov > moved {
							moved = mov
						}
					}
				}

				audioPlay(moved)

				if agame.Count == 0 {
					if agame.Winner() {
						s.PostEvent(tcell.NewEventInterrupt(true))
					}
				} else if moved != game.None {
					agame.Seq = 0
				}

				checkScreen(s, cx, cy, game.None)
			} else if crune == 'P' || crune == 'p' { // auto play
				s.PostEvent(tcell.NewEventInterrupt(false))
			}
		case *tcell.EventMouse:
			cx, cy = ev.Position()
			pressed := ev.Buttons()&tcell.ButtonMask(0xff) != tcell.ButtonNone
			_, _, mov := checkScreen(s, cx, cy, ops[pressed])
			if pressed {
				audioPlay(mov)

				if agame.Count == 0 {
					if agame.Winner() {
						s.PostEvent(tcell.NewEventInterrupt(true))
					}
				}
			}

		case *tcell.EventInterrupt:
			winner := ev.Data().(bool)
			count := agame.Count

			for y := 1; y < agame.Height-1; y++ {
				for x := 1; x < agame.Width-1; x++ {
					x, y := agame.ScreenCoords(0, 0, x, y)
					agame.Update(x, y, game.Remove)
				}
			}

			if count == agame.Count { // no changes
				break
			}

			checkScreen(s, cx, cy, game.None)

			if agame.Count > 0 {
				if !winner {
					audioPlay(game.Shuffle)
					agame.Shuffle()
				}

				time.AfterFunc(300*time.Millisecond, func() { s.PostEvent(ev) })
			}
		}
	}
}
