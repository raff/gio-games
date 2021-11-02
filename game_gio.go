package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"

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

var (
	//go:embed assets/up-arrow.png
	pngUp []byte

	//go:embed assets/dot.png
	pngDot []byte

	bgColor = color.NRGBA{0, 0, 32, 255}

	gDirs [5]image.Image
	gDot  image.Image
	cell  image.Point

	canvas draw.Image
	wopts  []app.Option
)

func setTitle(w *app.Window, title string) {
	if title == "" {
		title = fmt.Sprintf("moves=%v remain=%v removed=%v seq=%v/%v score=%v",
			game.Moves, game.Count, game.Removed, game.Seq, game.MaxSeq, game.Score)
	}
	wopts[0] = app.Title(title)
	w.Option(wopts...)
}

func updateScore(printed bool) bool {
	if !printed {
		if newscore := scores.Update(&game); newscore != nil {
			fmt.Printf("New best score: moves=%v seq=%v score=%v\n",
				newscore.Moves, newscore.MaxSeq, newscore.Score)
		} else {
			fmt.Printf("Score: moves=%v seq=%v score=%v\n",
				game.Moves, game.MaxSeq, game.FinalScore)
		}
	}

	return true
}

func gioGame(terminate func()) {
	if img, err := png.Decode(bytes.NewBuffer(pngUp)); err != nil {
		log.Fatal(err)
	} else {
		cell = img.Bounds().Size()

		gDirs[Empty] = imaging.New(cell.X, cell.Y, bgColor)
		gDirs[Up] = img
		gDirs[Left] = imaging.Rotate90(gDirs[Up])
		gDirs[Down] = imaging.Rotate90(gDirs[Left])
		gDirs[Right] = imaging.Rotate90(gDirs[Down])
	}

	if img, err := png.Decode(bytes.NewBuffer(pngDot)); err != nil {
		log.Fatal(err)
	} else {
		gDot = img
	}

	wopts = []app.Option{
		app.Title("Arrows"), // title is first option
		app.Size(unit.Px(float32(gameWidth*cell.X)), unit.Px(float32(gameHeight*cell.Y))),
		app.MinSize(unit.Px(float32(gameWidth*cell.X)), unit.Px(float32(gameHeight*cell.Y))),
		app.MaxSize(unit.Px(float32(gameWidth*cell.X)), unit.Px(float32(gameHeight*cell.Y))),
	}

	go func() {
		w := app.NewWindow(wopts...)
		loop(w)
		terminate()
	}()
	app.Main()
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func playturn(w *app.Window, title bool) (bool, bool) {
	moved := Invalid

	for y := 1; y < game.Height-1; y++ {
		for x := 1; x < game.Width-1; x++ {
			x, y := game.ScreenCoords(0, 0, x, y)
			_, _, mov := game.Update(x, y, Remove)
			if mov > moved {
				moved = mov
			}
		}
	}

	audioPlay(moved)

	if title {
		setTitle(w, "")
	}

	if game.Count != 0 && moved != None {
		game.Seq = 0
	}

	w.Invalidate()
	return moved != None, game.Count == 0
}

func loop(w *app.Window) {
	var ops op.Ops

	game.Setup(gameWidth, gameHeight, cell.X, cell.Y)

	gw := gameWidth * cell.X
	gh := gameHeight * cell.Y

	cx, cy := 1, 1

	gameover := false
	autoplay := false
	dotscreen := false
	printscore := false

	for e := range w.Events() {
		switch e := e.(type) {
		case system.DestroyEvent:
			if e.Err != nil {
				fmt.Println(e.Err)
			}
			return

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			if gw > e.Size.X || gh > e.Size.Y {
				gw = min(gw, e.Size.X)
				gh = min(gh, e.Size.Y)

				gameWidth = gw / cell.X
				gameHeight = gh / cell.Y

				game.Setup(gameWidth, gameHeight, cell.X, cell.Y)
			}

			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				pressed := false

				if gameover || autoplay {
					if _, done := playturn(w, !gameover); done {
						gameover = true
						printscore = updateScore(printscore)

						if game.Winner() {
							setTitle(w, "")
						} else {
							setTitle(w, "You Win!")
							dotscreen = true
						}
					} else if !gameover && autoplay {
						audioPlay(Shuffle)
						game.Shuffle(shuffleDir)
						setTitle(w, "")
					}
				} else {
					// Handle any input from a pointer.
					for _, ev := range gtx.Events(gDirs) {
						if ev, ok := ev.(pointer.Event); ok {
							if ev.Type == pointer.Press {
								_, _, mov := game.Update(int(ev.Position.X), int(ev.Position.Y), Move)
								audioPlay(mov)

								if mov != Invalid {
									setTitle(w, "")
								}

								if game.Count == 0 {
									gameover = true
									printscore = updateScore(printscore)

									if game.Winner() {
										setTitle(w, "")
									} else {
										setTitle(w, "You Win!")
										dotscreen = true
									}

									w.Invalidate()
								}

								pressed = true
							} else { // Move
								x, y, dir := game.Peek(int(ev.Position.X), int(ev.Position.Y))
								if dir != Empty {
									cx, cy = x, y
								}
							}
						}
					}
				}

				// Register to listen for pointer events.
				pr := pointer.Rect(image.Rectangle{Max: e.Size}).Push(gtx.Ops)
				pointer.InputOp{Tag: gDirs, Types: pointer.Press | pointer.Move}.Add(gtx.Ops)
				pr.Pop()

				return render(gtx, gw, gh, cx, cy, pressed, dotscreen)
			})

			e.Frame(gtx.Ops)

		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case key.NameEscape, "Q", "X":
					w.Close()

				case key.NameUpArrow:
					sx, sy := game.ScreenCoords(0, 0, cx, cy-1)
					if _, _, dir := game.Peek(sx, sy); dir != InvalidDir {
						cy--
						w.Invalidate()
					}

				case key.NameDownArrow:
					sx, sy := game.ScreenCoords(0, 0, cx, cy+1)
					if _, _, dir := game.Peek(sx, sy); dir != InvalidDir {
						cy++
						w.Invalidate()
					}

				case key.NameLeftArrow:
					sx, sy := game.ScreenCoords(0, 0, cx-1, cy)
					if _, _, dir := game.Peek(sx, sy); dir != InvalidDir {
						cx -= 1
						w.Invalidate()
					}

				case key.NameRightArrow:
					sx, sy := game.ScreenCoords(0, 0, cx+1, cy)
					if _, _, dir := game.Peek(sx, sy); dir != InvalidDir {
						cx += 1
						w.Invalidate()
					}

				case key.NameSpace:
					x, y := game.ScreenCoords(0, 0, cx, cy)
					_, _, mov := game.Update(x, y, Move)
					audioPlay(mov)

					if mov != Invalid {
						setTitle(w, "")
					}

					if game.Count == 0 {
						gameover = true
						printscore = updateScore(printscore)

						if game.Winner() {
							setTitle(w, "")
						} else {
							setTitle(w, "You Win!")
							dotscreen = true
						}
					}

					w.Invalidate()

				case "U": // undo
					if _, _, ok := game.Undo(); ok {
						audioPlay(Undo)
						setTitle(w, "")
						w.Invalidate()
					}
				case "R": // reset
					audioPlay(Undo)
					game.Setup(gameWidth, gameHeight, cell.X, cell.Y)
					setTitle(w, "Arrows")
					gameover = false
					dotscreen = false
					autoplay = false
					printscore = false
					w.Invalidate()

				case "S": // reshuffle
					audioPlay(Shuffle)
					game.Shuffle(shuffleDir)
					setTitle(w, "")
					w.Invalidate()

				case "H": // help: remove all "free" arrows
					_, gameover = playturn(w, true)
					if gameover {
						printscore = updateScore(printscore)

						if game.Winner() {
							setTitle(w, "")
						} else {
							setTitle(w, "You Win!")
							dotscreen = true
						}

						w.Invalidate()
					}

				case "P": // autoplay
					autoplay = true
					w.Invalidate()
				}
			}
		}
	}
}

func render(gtx layout.Context, gw, gh, px, py int, pressed, dotscreen bool) layout.Dimensions {
	if canvas == nil {
		canvas = imaging.New(gw, gh, bgColor)
	} else {
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{bgColor}, image.ZP, draw.Src)
	}

	for y, row := range game.Screen {
		for x, col := range row {
			im := gDirs[col]

			if dotscreen {
				im = gDot
			} else if !pressed && px == x && py == y {
				if col == Empty {
					im = gDot
				} else {
					im = imaging.Invert(im)
				}
			}

			draw.Draw(canvas,
				im.Bounds().Add(image.Point{x * cell.X, y * cell.Y}),
				im, image.Point{}, draw.Over)
		}
	}

	canvasOp := paint.NewImageOp(canvas)
	img := widget.Image{Src: canvasOp}
	img.Scale = 1 / float32(gtx.Px(unit.Dp(1)))

	return img.Layout(gtx)
}
