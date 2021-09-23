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

	bgColor = color.NRGBA{0, 0, 32, 255}

	gDirs [5]image.Image
	cell  image.Point

	canvas draw.Image
	gw, gh int

	wopts []app.Option
)

func setTitle(w *app.Window, msg string, args ...interface{}) {
	wopts[0] = app.Title(fmt.Sprintf(msg, args...))
	w.Option(wopts...)
}

func gioGame() {
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

	game.Setup(gameWidth, gameHeight, cell.X, cell.Y)

	gw = gameWidth * cell.X
	gh = gameHeight * cell.Y

	wopts = []app.Option{
		app.Title("Arrows"), // title is first option
		app.Size(unit.Px(float32(gameWidth*cell.X)), unit.Px(float32(gameHeight*cell.Y))),
		app.MinSize(unit.Px(float32(gameWidth*cell.X)), unit.Px(float32(gameHeight*cell.Y))),
		app.MaxSize(unit.Px(float32(gameWidth*cell.X)), unit.Px(float32(gameHeight*cell.Y))),
	}

	go func() {
		w := app.NewWindow(wopts...)
		loop(w)
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) {
	// th := material.NewTheme(gofont.Collection())
	var ops op.Ops

	for e := range w.Events() {
		switch e := e.(type) {
		case system.DestroyEvent:
			if e.Err != nil {
				fmt.Println(e.Err)
			}
			return

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			// Handle any input from a pointer.
			for _, ev := range gtx.Events(gDirs) {
				if ev, ok := ev.(pointer.Event); ok {
					_, _, mov := game.Update(int(ev.Position.X), int(ev.Position.Y), Move)
					audioPlay(mov)

					if mov != Invalid {
						setTitle(w, "moves=%v remain=%v removed=%v seq=%v",
							game.Moves, game.Count, game.Removed, game.Seq)
					}

					if game.Count == 0 {
						if !game.Winner() {
							setTitle(w, "You Win!")
						} else {
							w.Invalidate()
						}
					}

				}
			}
			// Register to listen for pointer Drag events.
			pointer.Rect(image.Rectangle{Max: e.Size}).Add(gtx.Ops)
			pointer.InputOp{Tag: gDirs, Types: pointer.Press}.Add(gtx.Ops)

			render(gtx, e.Size)
			e.Frame(gtx.Ops)

		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case key.NameEscape, "Q", "X":
					w.Close()

				case "U": // undo
					if _, _, ok := game.Undo(); ok {
						audioPlay(Undo)
						setTitle(w, "moves=%v remain=%v removed=%v seq=%v",
							game.Moves, game.Count, game.Removed, game.Seq)
						w.Invalidate()
					}
				case "R": // reset
					audioPlay(Undo)
					game.Setup(gameWidth, gameHeight, cell.X, cell.Y)
					setTitle(w, "Arrows")
					w.Invalidate()

				case "S": // reshuffle
					audioPlay(Shuffle)
					game.Shuffle()
					setTitle(w, "moves=%v remain=%v removed=%v seq=%v",
						game.Moves, game.Count, game.Removed, game.Seq)
					w.Invalidate()

				case "H": // help: remove all "free" arrows
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

					setTitle(w, "moves=%v remain=%v removed=%v seq=%v",
						game.Moves, game.Count, game.Removed, game.Seq)

					if game.Count == 0 {
						if !game.Winner() {
							setTitle(w, "You Win!")
						}
					} else if moved != None {
						game.Seq = 0
					}

					w.Invalidate()
				}
			}
		}
	}
}

func render(gtx layout.Context, sz image.Point) {
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if canvas == nil {
			canvas = imaging.New(gw, gh, bgColor)
		} else {
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{bgColor}, image.ZP, draw.Src)
		}

		for y, row := range game.Screen {
			for x, col := range row {
				im := gDirs[col]

				draw.Draw(canvas,
					im.Bounds().Add(image.Point{x * cell.X, y * cell.Y}),
					im, image.Point{}, draw.Over)
			}
		}

		canvasOp := paint.NewImageOp(canvas)
		img := widget.Image{Src: canvasOp}
		img.Scale = float32(sz.X) / float32(gtx.Px(unit.Dp(float32(sz.X))))

		return img.Layout(gtx)
	})
}
