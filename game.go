package main

import (
	"math/rand"
	"time"
)

// Arrow directions
type Dir int8

type Updates int8

const (
	InvalidDir = Dir(-1)
	Empty      = Dir(0)
	Up         = Dir(1)
	Down       = Dir(2)
	Left       = Dir(3)
	Right      = Dir(4)

	DirCount = 4

	Invalid = Updates(0) // invalid coordinates
	None    = Updates(1) // cannot move
	Move    = Updates(2) // move arrow/moved
	Remove  = Updates(3) // remove arrow/removed

	// these is actually only used for playing sound effects
	Shuffle = Updates(-1)
	Undo    = Updates(-2)
)

type Cell struct {
	X int
	Y int
	D Dir
}

type CellMoves struct {
	Cells   []Cell
	Count   int
	Removed bool
}

type Game struct {
	Screen     [][]Dir
	Width      int
	Height     int
	Count      int
	Removed    int
	Moves      int
	Seq        int
	MaxSeq     int
	Score      int
	FinalScore int
	Completed  bool

	cellwidth  int
	cellheight int

	stack []*CellMoves
}

func (g *Game) Push(count int, removed bool, moves []Cell) {
	g.stack = append(g.stack, &CellMoves{Cells: moves, Count: count, Removed: removed})
}

func (g *Game) Pop() (cm *CellMoves) {
	l := len(g.stack)

	if l == 0 {
		return nil
	}

	cm, g.stack = g.stack[l-1], g.stack[:l-1]
	return
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
	g.Seq = 0
	g.MaxSeq = 0
	g.Score = 0
	g.FinalScore = 0
	g.Completed = false

	g.cellwidth = cw
	g.cellheight = ch
	g.stack = g.stack[:0]

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

	g.simplify()
}

//
// shuffle arrows
// (actually replace/rotate arrows where present)
//
func (g *Game) Shuffle(dir Dir) {
	g.Count = 0
	g.Seq = 0

	for y, row := range g.Screen {
		for x, col := range row {
			if col != Empty {
				g.Count++

				switch dir {
				case Up, Left:
					// rotate left
					switch g.Screen[y][x] {
					case Up:
						g.Screen[y][x] = Left
					case Down:
						g.Screen[y][x] = Right
					case Left:
						g.Screen[y][x] = Down
					case Right:
						g.Screen[y][x] = Up
					}

				case Down, Right:
					// rotate right
					switch g.Screen[y][x] {
					case Up:
						g.Screen[y][x] = Right
					case Down:
						g.Screen[y][x] = Left
					case Left:
						g.Screen[y][x] = Up
					case Right:
						g.Screen[y][x] = Down
					}

				default:
					// random shuffle
					var newdir Dir
					for newdir = g.Screen[y][x]; newdir == g.Screen[y][x]; newdir = Dir(rand.Intn(DirCount) + 1) { // 0 is Empty
						// try again
					}

					g.Screen[y][x] = newdir
				}
			}
		}
	}

	if dir == Empty {
		g.simplify()
	}

	g.stack = g.stack[:0]
}

func (g *Game) simplify() {
	opposite := map[Dir]Dir{
		Up:    Down,
		Down:  Up,
		Left:  Right,
		Right: Left,
	}

	for y, row := range g.Screen {
		for x, col := range row {
			if col == Empty {
				continue
			}

			for c := 0; c < DirCount; c++ {
				opp := opposite[col]

				if g.Screen[y][x-1] == opp || g.Screen[y][x+1] == opp || g.Screen[y-1][x] == opp || g.Screen[y+1][x] == opp {
					switch col {
					case Up:
						col = Left
					case Down:
						col = Right
					case Left:
						col = Down
					case Right:
						col = Up
					}

					continue
				}

				break
			}

			g.Screen[y][x] = col

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

func (g *Game) Peek(x, y int) (int, int, Dir) {
	if cx, cy, ok := g.Coords(x, y); ok {
		return cx, cy, g.Screen[cy][cx]
	}

	return -1, -1, InvalidDir
}

//
// update game based on screen coordinates
// returns game coordinates (and false if outside of boundaries)
//
// remove: remove arrows at x,y
// move: if not out of boundary move arrow to last empty position
//
func (g *Game) Update(x, y int, op Updates) (cx, cy int, res Updates) {
	var ok bool

	cx, cy, ok = g.Coords(x, y)
	if !ok {
		return -1, -1, Invalid
	}

	res = None

	if op > None {
		var px, py int
		var cells, empty []Cell

		curdir := g.Screen[cy][cx]

		update := func(removing bool) (ret Updates) {
			lc := len(cells)
			le := len(empty)

			if removing { // got to the end, remove current arrow
				if !g.Completed {
					for _ = range cells {
						g.Count--
						g.Removed++
						g.Seq++
						g.Score += g.Seq
						if g.Seq > g.MaxSeq {
							g.MaxSeq = g.Seq
						}
					}
				}
				le = 0
				ret = Remove
			} else { // partial move
				if op != Move {
					return None // we requested full move
				}

				if le > lc {
					empty = empty[le-lc:]
					le = len(empty)
				}

				ret = Move
			}

			for _, c := range cells[:lc] {
				g.Screen[c.Y][c.X] = Empty // remove from old position
			}

			if le > 0 {
				cells = append(cells, empty...)

				for _, c := range cells[len(cells)-lc:] {
					g.Screen[c.Y][c.X] = curdir // move into new position
				}
			}

			g.Push(lc, removing, cells)
			if !g.Completed {
				g.Moves++
			}
			return
		}

		switch curdir {
		case Up:
			for py = cy; py > 0 && g.Screen[py][cx] == curdir; py-- {
				cells = append(cells, Cell{X: cx, Y: py, D: curdir})
			}

			for ppy := py; ppy >= 0 && g.Screen[ppy][cx] == Empty; ppy-- {
				py = ppy
				empty = append(empty, Cell{X: cx, Y: py, D: Empty})
			}

			if g.Screen[py][cx] != Empty {
				return
			}

			res = update(py == 0)

		case Down:
			for py = cy; py < g.Height-1 && g.Screen[py][cx] == curdir; py++ {
				cells = append(cells, Cell{X: cx, Y: py, D: curdir})
			}

			for ppy := py; ppy <= g.Height-1 && g.Screen[ppy][cx] == Empty; ppy++ {
				py = ppy
				empty = append(empty, Cell{X: cx, Y: py, D: Empty})
			}

			if g.Screen[py][cx] != Empty {
				return
			}

			res = update(py == g.Height-1)

		case Left:
			for px = cx; px > 0 && g.Screen[cy][px] == curdir; px-- {
				cells = append(cells, Cell{X: px, Y: cy, D: curdir})
			}

			for ppx := px; ppx >= 0 && g.Screen[cy][ppx] == Empty; ppx-- {
				px = ppx
				empty = append(empty, Cell{X: px, Y: cy, D: Empty})
			}

			if g.Screen[cy][px] != Empty {
				return
			}

			res = update(px == 0)

		case Right:
			for px = cx; px < g.Width-1 && g.Screen[cy][px] == curdir; px++ {
				cells = append(cells, Cell{X: px, Y: cy, D: curdir})
			}

			for ppx := px; ppx <= g.Width-1 && g.Screen[cy][ppx] == Empty; ppx++ {
				px = ppx
				empty = append(empty, Cell{X: px, Y: cy, D: Empty})
			}

			if g.Screen[cy][px] != Empty {
				return
			}

			res = update(px == g.Width-1)
		}
	}

	return
}

func (g *Game) Undo() (cx, cy int, ok bool) {
	if cm := g.Pop(); cm != nil {
		for _, m := range cm.Cells {
			cx, cy = m.X, m.Y
			g.Screen[m.Y][m.X] = m.D
		}

		if !g.Completed {
			g.Moves--

			if cm.Removed {
				g.Count += cm.Count
				g.Removed -= cm.Count
				if g.Seq > 0 {
					for n := cm.Count; n > 0; n-- {
						if g.Seq == g.MaxSeq {
							g.MaxSeq--
						}
						g.Score -= g.Seq
						g.Seq--
					}
				}
			}
		}

		return cx, cy, true
	}

	return -1, -1, false
}

var WinBanner = [][]Dir{
	{Down, Down, Up, Up, Down, Down, Up, Up, Down, Down, Down, Down, Up, Up, Down, Down, Up, Up, Down, Down},
	{Up, Up, Up, Up, Up, Up, Up, Down, Up, Up, Up, Up, Down, Up, Up, Up, Up, Up, Up, Up},
	{Up, Up, Up, Up, Up, Up, Up, Up, Up, Empty, Empty, Up, Up, Up, Up, Up, Up, Up, Up, Up},
	{Down, Up, Down, Down, Up, Empty, Up, Up, Up, Empty, Empty, Up, Up, Up, Up, Up, Up, Up, Up, Up},
	{Left, Left, Up, Up, Empty, Empty, Up, Up, Up, Empty, Empty, Up, Up, Up, Up, Up, Up, Up, Up, Up},
	{Left, Left, Up, Up, Empty, Empty, Up, Up, Up, Down, Down, Up, Up, Up, Up, Up, Down, Down, Up, Up},
	{Left, Left, Up, Up, Empty, Empty, Up, Empty, Up, Up, Up, Up, Empty, Empty, Empty, Up, Up, Up, Up, Right},
	{Left, Left, Left, Left, Left, Left, Left, Left, Left, Left, Right, Right, Right, Right, Right, Right, Right, Right, Right, Right},
	{Down, Down, Down, Empty, Empty, Empty, Down, Down, Down, Down, Down, Down, Down, Down, Empty, Empty, Empty, Down, Down, Right},
	{Down, Up, Up, Empty, Empty, Empty, Up, Up, Down, Up, Up, Down, Up, Up, Empty, Empty, Empty, Up, Up, Right},
	{Down, Up, Up, Empty, Empty, Empty, Up, Up, Down, Up, Up, Down, Up, Up, Down, Empty, Empty, Up, Up, Right},
	{Down, Up, Up, Empty, Empty, Empty, Up, Up, Down, Up, Up, Down, Up, Up, Up, Down, Empty, Up, Up, Right},
	{Down, Up, Up, Empty, Down, Empty, Up, Up, Down, Up, Up, Down, Up, Up, Down, Up, Down, Up, Up, Right},
	{Down, Up, Up, Down, Up, Down, Up, Up, Down, Up, Up, Down, Up, Up, Down, Down, Up, Up, Up, Right},
	{Down, Down, Up, Up, Down, Up, Up, Down, Down, Up, Up, Down, Up, Up, Down, Down, Down, Up, Up, Right},
}

func (g *Game) Winner() bool {
	g.Completed = true

	ww, hw := len(WinBanner[0]), len(WinBanner)

	if ww >= g.Width || hw >= g.Height {
		return false
	}

	ww = (g.Width - ww) / 2
	hw = (g.Height - hw) / 2

	g.Count = 0

	for y, row := range WinBanner {
		for x, col := range row {
			g.Screen[hw+y][ww+x] = col
			if col != Empty {
				g.Count++
			}
		}
	}

	return true
}

type ScoreInfo struct {
	Moves  int
	MaxSeq int
	Score  int
}

type Scores map[int][]ScoreInfo

func scoreKey(w, h int) int {
	return w*1000 + h
}

func (sc Scores) Update(g *Game) *ScoreInfo {
	n := g.Removed - g.Moves
	g.FinalScore = g.Score + (n * n / 2)

	info := ScoreInfo{Moves: g.Moves, MaxSeq: g.MaxSeq, Score: g.FinalScore}

	key := scoreKey(g.Width, g.Height)
	ss := sc[key]
	if ss == nil { // first entry
		sc[key] = []ScoreInfo{info}
		return &info
	}

	for i, si := range ss {
		if g.FinalScore > si.Score {
			ss = append(ss[:i+1], ss[i:]...)
			ss[i] = info
			if len(ss) > 10 {
				ss = ss[:10]
			}
			sc[key] = ss
			if i == 0 {
				return &info
			}

			return nil
		}
	}

	if len(ss) < 10 {
		ss = append(ss, info)
		sc[key] = ss
	}

	return nil
}

func (s Scores) Get(width, height int) []ScoreInfo {
	key := scoreKey(width, height)
	return s[key]
}

var game Game
var scores = Scores{}
