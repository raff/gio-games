package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"sort"
	"time"

	"math/rand"

	_ "embed"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/disintegration/imaging"
)

const (
	hcount = 10
	vcount = 4

	border = 8
)

var (
	//go:embed assets/tiles.png
	pngTiles []byte

	tiles []draw.Image

	borderColor = color.NRGBA{80, 80, 80, 255}

	canvas draw.Image

	tw, th int // game tile width, height
	gw, gh int // number of horizontal and vertical tiles in game
	ww, wh int // window width and height

	cards [][]int // gw columns of gh tiles (card indices)

	mcount = 2

	curmatches   = 0
	maxmatches   = 0
	totalmatches = 0
	shuffles     = 0

	mpoints = 1
	spoints = 1
	score   = 0

	wopts []app.Option
)

func initGame() {
	var lcards []int

	if len(tiles) == 0 {
		img, err := png.Decode(bytes.NewBuffer(pngTiles))
		if err != nil {
			log.Fatal(err)
		}

		isz := img.Bounds().Size()
		hsize := isz.X / hcount
		vsize := isz.Y / vcount

		tw = hsize / 2
		th = vsize / 2

		//card_loop:
		for v, y := 0, 0; v < vcount; v++ {
			for h, x := 0, 0; h < hcount; h++ {
				card := v*hcount + h

				tile := imaging.New(tw, th, borderColor)
				im := imaging.Crop(img, image.Rect(x, y, x+hsize, y+vsize))
				im = imaging.Resize(im, tw-border, th-border-border, imaging.Box)
				draw.Draw(tile,
					im.Bounds().Add(image.Point{border, border}),
					im, image.Point{}, draw.Over)
				tiles = append(tiles, tile)

				lcards = append(lcards, card)
				lcards = append(lcards, card)
				lcards = append(lcards, card)
				lcards = append(lcards, card)
				lcards = append(lcards, card)
				lcards = append(lcards, card)

				x += hsize
			}

			y += vsize
		}

		gw, gh = factors(len(lcards))
		ww, wh = gw*tw, (gh+1)*th/2

		cards = make([][]int, gw)

		for i := range cards {
			cards[i] = make([]int, gh)
		}
	}

	cols := 0
	ci := -1

	if len(lcards) == 0 {
		for i, col := range cards {
			lcards = append(lcards, col...)

			if len(col) > 0 {
				cols++
				ci = i
			}
		}
	}

	if cols == 1 {
		// only one column left
		// make it one row

		cards[ci] = nil

		for i, c := range lcards {
			cards[i] = []int{c}
		}
	} else {
		rand.Shuffle(len(lcards), func(i, j int) {
			if lcards[i] != -1 && lcards[j] != -1 {
				lcards[i], lcards[j] = lcards[j], lcards[i]
			}
		})

		i := 0

		for x, col := range cards {
			for y := range col {
				cards[x][y] = lcards[i]
				i++
			}
		}
	}

	canvas = imaging.New(ww, wh, borderColor)
	drawCards(nil)
}

func drawCards(revs map[int]bool) {
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{borderColor}, image.ZP, draw.Src)

	for x, col := range cards {
		for y, card := range col {
			im := tiles[card]
			ci := gameIndex(x, y)

			if revs != nil && revs[ci] {
				im = imaging.Invert(im)
			}

			draw.Draw(canvas,
				im.Bounds().Add(image.Point{x * tw, y * th / 2}),
				im, image.Point{}, draw.Over)
		}
	}
}

func cardIndex(x, y int) (int, int, int) {
	x /= tw
	y /= (th / 2)

	//log.Println("cardIndex", x, y)
	if c := playable(x, y); c >= 0 {
		return x, y, c
	}

	return -1, -1, -1
}

func playable(x, y int) int {
	if x >= 0 && x < gw && y >= 0 && y < gh {
		col := cards[x]
		if y == len(col)-1 { // last valid card in a column
			return col[y]
		}
	}

	return -1
}

func factors(n int) (int, int) {
	for i := n - 1; i > 1; i-- {
		if n%i == 0 {
			m1 := i
			m2 := n / i

			if m2 >= m1 {
				return m1, m2
			}
		}
	}

	return n, 1
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func getScore() string {
	return fmt.Sprintf("Tris - matches:%v  max.matches:%v  total:%v  shuffles:%v  score:%v  (%v)",
		curmatches, maxmatches, totalmatches, shuffles, score, spoints)
}

func setTitle(w *app.Window, title string) {
	wopts[0] = app.Title(title)
	w.Option(wopts...)
}

func main() {
	rand.Seed(time.Now().Unix())

	initGame()
	//log.Println(factors(len(cards)))

	wopts = []app.Option{
		app.Title("Tris"),
		app.Size(unit.Px(float32(ww)), unit.Px(float32(wh))),
		app.MinSize(unit.Px(float32(ww)), unit.Px(float32(wh))),
		app.MaxSize(unit.Px(float32(ww)), unit.Px(float32(wh))),
	}

	go func() {
		w := app.NewWindow(wopts...)
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func gameIndex(x, y int) int {
	return y*gw + x
}

func gameCoord(gi int) (int, int) {
	y := gi / gw
	x := gi % gw
	return x, y
}

func loop(w *app.Window) error {
	var ops op.Ops

	var matches map[int]bool
	match := -1

	autoplay := false

	defer func() {
		fmt.Println(getScore())
	}()

	playCard := func(x, y int) {
		card := playable(x, y)
		ci := gameIndex(x, y)

		if match != card {
			match = card
			matches = map[int]bool{ci: true}
		} else {
			matches[ci] = true
		}

		if len(matches) == mcount {
			for gi, _ := range matches {
				x, y := gameCoord(gi)
				cards[x] = cards[x][:y]
			}

			matches = nil
			match = -1

			curmatches++
			totalmatches++
			score += spoints
			if curmatches > maxmatches {
				maxmatches = curmatches
			}
			setTitle(w, getScore())

			lc := 0

			for _, col := range cards {
				lc += len(col)
			}

			if lc == 0 {
				w.Close()
			}
		}

		drawCards(matches)
		w.Invalidate()
	}

	shuffle := func() {
		initGame()

		shuffles++
		mpoints += curmatches
		spoints = int((float32(mpoints) + 0.5) / float32(shuffles))
		if spoints == 0 {
			spoints = 1
		}

		match = -1
		curmatches = 0
		setTitle(w, getScore())
	}

	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err

		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case key.NameEscape, "Q", "X":
					w.Close()
				case "R":
					shuffle()
					w.Invalidate()
				case "A":
					autoplay = !autoplay
					fmt.Println("autoplay:", autoplay)
					w.Invalidate()
				}
			}

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			if autoplay {
				playcards := make([]struct{ y, c int }, gw)
				for x, col := range cards {
					y := len(col) - 1
					c := playable(x, y)
					playcards[x] = struct{ y, c int }{y, c}
				}

				sort.Slice(playcards, func(i, j int) bool {
					return playcards[i].c >= playcards[j].c
				})

				if playcards[0].c == -1 {
					fmt.Println("no valid cards")
					autoplay = false
					continue
				}

				matched := false
				for x := 0; x < len(playcards)-1; x++ {
					if playcards[x+0].c >= 0 && playcards[x+0].c == playcards[x+1].c {
						playCard(x+0, playcards[x+0].y)
						playCard(x+1, playcards[x+1].y)
						matched = true
						break
					}
				}

				if !matched {
					shuffle()
					w.Invalidate()
				}
			} else {
				for _, ev := range gtx.Events("tris") {
					if ev, ok := ev.(pointer.Event); ok {
						if ev.Type == pointer.Press {
							x, y, c := cardIndex(int(ev.Position.X), int(ev.Position.Y))
							if c >= 0 {
								playCard(x, y)
							}
						}
					}
				}
			}

			canvasOp := paint.NewImageOp(canvas)
			img := widget.Image{Src: canvasOp}
			img.Scale = 1 / float32(gtx.Px(unit.Dp(1)))
			img.Layout(gtx)

			pr := pointer.Rect(image.Rectangle{Max: e.Size}).Push(gtx.Ops)
			pointer.InputOp{Tag: "tris", Types: pointer.Press}.Add(gtx.Ops)
			pr.Pop()

			e.Frame(gtx.Ops)

			if autoplay {
				w.Invalidate()
			}
		}
	}
}
