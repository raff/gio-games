//go:build ignore

package main

import (
	"bytes"
	"flag"
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

	dirs [5]image.Image
	cell image.Point

	canvas draw.Image

	width  = 20
	height = 20

	game Game

	wopts []app.Option
)

func setTitle(w *app.Window, msg string, args ...interface{}) {
	wopts[0] = app.Title(fmt.Sprintf(msg, args...))
	w.Option(wopts...)
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

	if img, err := png.Decode(bytes.NewBuffer(pngUp)); err != nil {
		log.Fatal(err)
	} else {
		cell = img.Bounds().Size()

		dirs[Empty] = imaging.New(cell.X, cell.Y, bgColor)
		dirs[Up] = img
		dirs[Left] = imaging.Rotate90(dirs[Up])
		dirs[Down] = imaging.Rotate90(dirs[Left])
		dirs[Right] = imaging.Rotate90(dirs[Down])
	}

	game.Setup(width, height, cell.X, cell.Y)

	wopts = []app.Option{
		app.Title("Arrows"), // title is first option
		app.Size(unit.Px(float32(width*cell.X)), unit.Px(float32(height*cell.Y))),
		app.MinSize(unit.Px(float32(width*cell.X)), unit.Px(float32(height*cell.Y))),
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
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			// Handle any input from a pointer.
			for _, ev := range gtx.Events(dirs) {
				if ev, ok := ev.(pointer.Event); ok {
					_, _, mov := game.Update(int(ev.Position.X), int(ev.Position.Y), Move)
					audioPlay(mov)

					if mov != Invalid {
						setTitle(w, "moves=%v remain=%v removed=%v seq=%v",
							game.Moves, game.Count, game.Removed, game.Seq)
					}
				}
			}
			// Register to listen for pointer Drag events.
			pointer.Rect(image.Rectangle{Max: e.Size}).Add(gtx.Ops)
			pointer.InputOp{Tag: dirs, Types: pointer.Press}.Add(gtx.Ops)

			render(gtx, e.Size)
			e.Frame(gtx.Ops)
		case system.DestroyEvent:
			if e.Err != nil {
				fmt.Println(e.Err)
			}
			return
		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case key.NameEscape, "Q", "X":
					w.Close()

				case "R": // reset
					audioPlay(Undo)
					game.Setup(width, height, cell.X, cell.Y)
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
	px := gtx.Metric.Px(unit.Dp(float32(sz.X)))
	py := gtx.Metric.Px(unit.Dp(float32(sz.Y)))

	if canvas == nil || canvas.Bounds().Size().X != px || canvas.Bounds().Size().Y != py {
		canvas = imaging.New(px, py, bgColor)
	} else {
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{bgColor}, image.ZP, draw.Src)
	}

	for y, row := range game.Screen {
		for x, col := range row {
			im := dirs[col]

			draw.Draw(canvas,
				im.Bounds().Add(image.Point{x * cell.X, y * cell.Y}),
				im, image.Point{}, draw.Over)
		}
	}

	canvasOp := paint.NewImageOp(canvas)
	img := widget.Image{Src: canvasOp}
	img.Scale = float32(sz.X) / float32(gtx.Px(unit.Dp(float32(sz.X))))
	img.Layout(gtx)
}
