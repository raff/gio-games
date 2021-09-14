package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell"
)

const (
	up    = '\u2b06' // '\u2191'
	down  = '\u2b07' // '\u2193'
	left  = '\u2b05' // '\u2190'
	right = '\u2b95' // '\u2192'
	empty = ' '

	sx = 1
	sy = 1
)

var (
	width  = 20
	height = 20

	dirs    = []rune{up, down, left, right}
	screen  [][]rune
	count   int
	removed int
	moves   int

	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	revStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorGreen)
)

func setupScreen() {
	screen = nil
	count = 0
	removed = 0
	moves = 0

	for i := 0; i < height; i++ {
		var line []rune

		for j := 0; j < width; j++ {
			cell := dirs[rand.Intn(len(dirs))]

			if i == 0 || i == height-1 || j == 0 || j == width-1 {
				// empty cell at the border, to make it easier to check if we can move
				cell = empty
			} else {
				count++
			}

			line = append(line, cell)
		}

		screen = append(screen, line)
	}
}

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
	x2 := x1 + len(screen[0])*2 + 1
	y2 := y1 + len(screen) + 1
	style := boxStyle

	// Fill screen
	for y, row := range screen {
		for x, col := range row {
			s.SetContent(x1+(2*x)+1, y1+y+1, col, nil, style)
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

func updateScreen(s tcell.Screen, x, y int, pressed bool) (int, int, bool) {
	if x > sx+2 && x < sx+(width*2)-1 && y > sy+1 && y < sy+height {
		cx := (x - sx - 2) / 2
		cy := y - sy - 1

		x = (cx * 2) + 2

		if pressed {
			var px, py int

			switch screen[cy][cx] {
			case up:
				for py = cy; py > 0 && screen[py-1][cx] == empty; py-- {
				}

				if py != cy {
					screen[cy][cx] = empty
					moves++

					if py == 0 {
						count--
						removed++
					} else {
						screen[py][cx] = up
					}
				}

			case down:
				for py = cy; py < height-1 && screen[py+1][cx] == empty; py++ {
				}

				if py != cy {
					screen[cy][cx] = empty
					moves++

					if py == height-1 {
						count--
						removed++
					} else {
						screen[py][cx] = down
					}
				}

			case left:
				for px = cx; px > 0 && screen[cy][px-1] == empty; px-- {
				}

				if px != cx {
					screen[cy][cx] = empty
					moves++

					if px == 0 {
						count--
						removed++
					} else {
						screen[cy][px] = left
					}
				}

			case right:
				for px = cx; px < width-1 && screen[cy][px+1] == empty; px++ {
				}

				if px != cx {
					screen[cy][cx] = empty
					moves++

					if px == width-1 {
						count--
						removed++
					} else {
						screen[cy][px] = right
					}
				}
			}
		}

		drawScreen(s)
		return cx, cy, true
	}

	return -1, -1, false
}

func main() {
	flag.IntVar(&width, "width", width, "screen width")
	flag.IntVar(&height, "height", height, "screen height")
	flag.Parse()

	if width <= 0 || height <= 0 {
		log.Fatal("invalid width or height")
	}

	width += 2  // add border
	height += 2 // to simplify boundary checks

	rand.Seed(time.Now().Unix())

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
	setupScreen()
	drawScreen(s)

	// Event loop
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

	for {
		// Update screen
		s.Show()

		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				quit()
			} else if ev.Key() == tcell.KeyCtrlL {
				s.Sync()
			} else if ev.Rune() == 'R' || ev.Rune() == 'r' {
				s.Clear()
				setupScreen()
				drawScreen(s)
			}
		case *tcell.EventMouse:
			x, y := ev.Position()

			button := ev.Buttons() & tcell.ButtonMask(0xff)

			msg := "                               "
			if _, _, ok := updateScreen(s, x, y, button != tcell.ButtonNone); ok {
				msg = fmt.Sprintf("moves=%v remain=%v removed=%v  ", moves, count, removed)
			}

			drawText(s, 1, height+3, len(msg)+1, height+3, boxStyle, msg)
		}
	}
}
