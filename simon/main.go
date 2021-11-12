package main

import (
	"image"
	"image/color"
	"log"
	"math/rand"
	"os"
	"time"

	"gioui.org/app" // app contains Window handling.
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/key" // key is used for keyboard events.
	"gioui.org/io/pointer"
	"gioui.org/io/system" // system is used for system events (e.g. closing the window).
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"gioui.org/x/outlay"
)

type (
	D = layout.Dimensions
	C = layout.Context
)

type Pad struct {
	button *widget.Clickable
	label  string
	color  color.NRGBA
}

func (pad Pad) Layout(gtx C, th *material.Theme, active bool) D {
	return material.Clickable(gtx, pad.button, func(gtx C) D {
		return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
			c := pad.color
			if !active {
				c = Darker(c)
			}

			dims := DrawRect(gtx, c, gtx.Constraints.Max, 20)
			layout.Center.Layout(gtx, material.H1(th, pad.label).Layout)
			return dims
		})
	})
}

func DrawRect(gtx C, background color.NRGBA, size image.Point, radii float32) D {
	bounds := f32.Rectangle{Max: f32.Pt(float32(size.X), float32(size.Y))}
	paint.FillShape(gtx.Ops, background, clip.UniformRRect(bounds, radii).Op(gtx.Ops))
	return layout.Dimensions{Size: size}
}

// Darker blends color towards a darker color.
func Darker(c color.NRGBA) (d color.NRGBA) {
	const r = 2 // darken ration
	return color.NRGBA{
		R: c.R / r,
		G: c.G / r,
		B: c.B / r,
		A: c.A,
	}
}

type Sequence struct {
	list   []int
	lindex int

	maxval  int
	current int

	timer *time.Timer
}

func (s *Sequence) Reset(add bool) {
	s.current = -1
	if add {
		s.lindex = -1
	} else {
		s.lindex = 0
	}
}

func (s *Sequence) Next() bool {
	if s.lindex == len(s.list) {
		return false
	}

	if s.lindex < 0 {
		s.lindex = 0
		s.list = append(s.list, rand.Intn(s.maxval))
	}

	s.current = s.list[s.lindex]
	s.lindex++
	return true
}

func (s *Sequence) HasNext() bool {
	return s.lindex < len(s.list)
}

func (s *Sequence) Current() (curr int) {
	curr, s.current = s.current, -1
	return
}

func (s *Sequence) Play(w *app.Window) {
	s.Stop()

	if s.Next() {
		w.Invalidate()

		s.timer = time.AfterFunc(playInterval, func() {
			s.Play(w)
		})
	}

	// log.Println(sequence)
}

func (s *Sequence) Stop() {
	if s.timer != nil {
		t := s.timer
		s.timer = nil
		t.Stop()
	}
}

var (
	ww = float32(800)
	wh = float32(600)

	playInterval = time.Second
	resetTime    = 500 * time.Millisecond

	pads = []Pad{
		{new(widget.Clickable), "1", color.NRGBA{A: 255, R: 0, G: 200, B: 0}},   // green
		{new(widget.Clickable), "2", color.NRGBA{A: 255, R: 255, G: 0, B: 0}},   // red
		{new(widget.Clickable), "3", color.NRGBA{A: 255, R: 255, G: 255, B: 0}}, // yellow
		{new(widget.Clickable), "4", color.NRGBA{A: 255, R: 0, G: 128, B: 255}}, // blue
	}

	sequence = Sequence{maxval: 4}
)

func main() {
	rand.Seed(time.Now().Unix())

	audioInit()

	go func() {
		w := app.NewWindow(
			app.Title("Simon"),
			app.Size(unit.Px(ww), unit.Px(wh)),
			app.MinSize(unit.Px(ww), unit.Px(wh)),
			app.MaxSize(unit.Px(ww), unit.Px(wh)),
		)
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) error {
	var ops op.Ops

	th := material.NewTheme(gofont.Collection())

	grid := outlay.Grid{Num: 2, Axis: layout.Horizontal}
	playSimon := true
	selected := -1

	sequence.Reset(true)
	sequence.Play(w)

	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			if playSimon {
				gtx = gtx.Disabled()
			}

			//log.Println(e)

			if playSimon { // the FrameEvent is from invalidate
				simon := sequence.Current()
				if simon >= 0 {
					selected = simon
					time.AfterFunc(resetTime, w.Invalidate) // and turn off
				} else if !sequence.HasNext() {
					log.Println("user play...")
					playSimon = false
					sequence.Reset(false)
				}
			}

			user := -1

			grid.Layout(gtx, len(pads), func(gtx C, i int) D {
				gtx.Constraints.Max.X = gtx.Constraints.Max.X / 2
				gtx.Constraints.Max.Y = int(wh) / 2

				pad := pads[i]

				if !playSimon {
					for _, ev := range gtx.Events(pad.label) {
						ev, _ := ev.(pointer.Event)

						if ev.Type == pointer.Press {
							user = i
						} else if ev.Type == pointer.Release {
							user = -1
						}
					}

					selected = user
				}

				if selected == i && !audioPlaying {
					log.Println("play", i)
					audioPlay(selected)
				}

				// Register to listen for pointer events.
				pr := pointer.Rect(image.Rectangle{Max: e.Size}).Push(gtx.Ops)
				pointer.InputOp{Tag: pad.label, Types: pointer.Press}.Add(gtx.Ops)
				pr.Pop()

				dims := pad.Layout(gtx, th, selected == i)
				pointer.CursorNameOp{Name: pointer.CursorPointer}.Add(gtx.Ops)
				return dims
			})

			if !playSimon && user >= 0 { // there are FrameEvents that are not from button clicks
				// if user >= 0 it was a button click
				simon := sequence.Current()
				if simon >= 0 {
					log.Println("simon", simon, "user", user)
					if simon != user {
						time.AfterFunc(playInterval, func() {
							audioPlay(audioBuzz)
							w.Close()
						})
					}
				}
				if !sequence.HasNext() {
					playSimon = true
					log.Println("simon play...")
					sequence.Reset(true)
					sequence.Play(w)
				}
			}

			for i := 0; i < len(pads); i++ {
				var pos image.Rectangle

				switch i {
				case 0:
					pos = image.Rect(0, 0, int(ww/2), int(wh/2))
				case 1:
					pos = image.Rect(int(ww/2), 0, int(ww), int(wh/2))
				case 2:
					pos = image.Rect(0, int(wh/2), int(ww/2), int(wh))
				case 3:
					pos = image.Rect(int(ww/2), int(wh/2), int(ww), int(wh))
				}

				// Register to listen for pointer events.
				pr := pointer.Rect(pos).Push(gtx.Ops)
				pointer.InputOp{Tag: pads[i].label, Types: pointer.Press | pointer.Release}.Add(gtx.Ops)
				pr.Pop()
			}

			e.Frame(gtx.Ops)
			selected = -1

		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case "1", "2", "3", "4":
					if !playSimon {
						selected = int(e.Name[0] - '1')
						w.Invalidate()
					}

				case "X", "Q":
					w.Close()
				}
			} else {
				selected = -1
				w.Invalidate()
			}
		}
	}
}
