package game

import (
	"math/rand"
	"time"
)

// Arrow directions
type Dir int8

const (
	Empty = Dir(0)
	Up    = Dir(1)
	Down  = Dir(2)
	Left  = Dir(3)
	Right = Dir(4)

	DirCount = 4
)

type Game struct {
	Screen  [][]Dir
	Width   int
	Height  int
	Count   int
	Removed int
	Moves   int

	cellwidth  int
	cellheight int
}

//
// setup game
//
func (g *Game) Setup(w, h, cw, ch int) {
	g.Screen = nil
	g.Width = w
	g.Height = h
	g.Count = 0
	g.Removed = 0
	g.Moves = 0

	g.cellwidth = cw
	g.cellheight = ch

	rand.Seed(time.Now().Unix())

	for i := 0; i < g.Height; i++ {
		var line []Dir

		for j := 0; j < g.Width; j++ {
			cell := Dir(rand.Intn(DirCount) + 1) // 0 is Empty

			if i == 0 || i == g.Height-1 || j == 0 || j == g.Width-1 {
				// empty cell at the border, to make it easier to check if we can move
				cell = Empty
			} else {
				g.Count++
			}

			line = append(line, cell)
		}

		g.Screen = append(g.Screen, line)
	}
}

//
// shuffle arrows
// (actually replace arrows where present)
//
func (g *Game) Shuffle() {
	for y, row := range g.Screen {
		for x, col := range row {
			if col != Empty {
				g.Screen[y][x] = Dir(rand.Intn(DirCount) + 1) // 0 is Empty
			}
		}
	}
}

//
// convert screen coordinates to game coordinates
//
// returns false if screen coordinates are outside game boundary
//
func (g *Game) Coords(x, y int) (int, int, bool) {
	x /= g.cellwidth
	y /= g.cellheight

	if x > 0 && x < g.Width-1 && y > 0 && y < g.Height-1 {
		return x, y, true
	}

	return -1, -1, false
}

//
// convert game coordinates to screen coordinates
//
// x,y: game coordinates
// sx,sy: screen offset
//
func (g *Game) ScreenCoords(sx, sy, x, y int) (int, int) {
	return sx + (x * g.cellwidth), sy + (y * g.cellheight)
}

//
// update game based on screen coordinates
// returns game coordinates (and false if outside of boundaries)
//
// remove: remove arrows at x,y
// move: if not out of boundary move arrow to last empty position
//
func (g *Game) Update(x, y int, remove, move bool) (cx, cy int, ok bool) {
	cx, cy, ok = g.Coords(x, y)
	if !ok {
		return
	}

	if remove {
		var px, py int

		switch g.Screen[cy][cx] {
		case Up:
			for py = cy; py > 0 && g.Screen[py-1][cx] == Empty; py-- {
			}

			if py != cy {
				if py == 0 {
					g.Count--
					g.Removed++
				} else {
					if !move {
						return
					}

					g.Screen[py][cx] = Up
				}

				g.Screen[cy][cx] = Empty
				g.Moves++

			}

		case Down:
			for py = cy; py < g.Height-1 && g.Screen[py+1][cx] == Empty; py++ {
			}

			if py != cy {
				if py == g.Height-1 {
					g.Count--
					g.Removed++
				} else {
					if !move {
						return
					}
					g.Screen[py][cx] = Down
				}

				g.Screen[cy][cx] = Empty
				g.Moves++

			}

		case Left:
			for px = cx; px > 0 && g.Screen[cy][px-1] == Empty; px-- {
			}

			if px != cx {
				if px == 0 {
					g.Count--
					g.Removed++
				} else {
					if !move {
						return
					}

					g.Screen[cy][px] = Left
				}

				g.Screen[cy][cx] = Empty
				g.Moves++
			}

		case Right:
			for px = cx; px < g.Width-1 && g.Screen[cy][px+1] == Empty; px++ {
			}

			if px != cx {
				if px == g.Width-1 {
					g.Count--
					g.Removed++
				} else {
					if !move {
						return
					}

					g.Screen[cy][px] = Right
				}

				g.Screen[cy][cx] = Empty
				g.Moves++
			}
		}
	}

	return
}
