package main

import "gobject/gtk-3.0"

type ButtonBoxesApp struct {
	window *gtk.Window
}

var ButtonBoxes ButtonBoxesApp

func (*ButtonBoxesApp) CreateBBox(orient gtk.Orientation, title string, spacing int, layout gtk.ButtonBoxStyle) *gtk.Frame {
	frame := gtk.NewFrame(title)
	bbox := gtk.NewButtonBox(orient)

	bbox.SetBorderWidth(5)
	frame.Add(bbox)

	bbox.SetLayout(layout)
	bbox.SetSpacing(spacing)

	var button *gtk.Button
	button = gtk.NewButtonFromStock(gtk.StockOk)
	bbox.Add(button)

	button = gtk.NewButtonFromStock(gtk.StockCancel)
	bbox.Add(button)

	button = gtk.NewButtonFromStock(gtk.StockHelp)
	bbox.Add(button)

	return frame
}

func (bb *ButtonBoxesApp) Do(mainwin *gtk.Window) *gtk.Window {
	if bb.window == nil {
		bb.window = gtk.NewWindow(gtk.WindowTypeToplevel)
		bb.window.SetTitle("Button Boxes")
		bb.window.Connect("destroy", func() { bb.window = nil })
		bb.window.SetBorderWidth(10)

		main_vbox := gtk.NewBox(gtk.OrientationVertical, 0)
		bb.window.Add(main_vbox)

		frame_horz := gtk.NewFrame("Horizontal Button Boxes")
		main_vbox.PackStart(frame_horz, true, true, 10)

		vbox := gtk.NewBox(gtk.OrientationVertical, 0)
		vbox.SetBorderWidth(10)
		frame_horz.Add(vbox)

		vbox.PackStart(
			bb.CreateBBox(gtk.OrientationHorizontal, "Spread",
				40, gtk.ButtonBoxStyleSpread), true, true, 0)
		vbox.PackStart(
			bb.CreateBBox(gtk.OrientationHorizontal, "Edge",
				40, gtk.ButtonBoxStyleEdge), true, true, 5)
		vbox.PackStart(
			bb.CreateBBox(gtk.OrientationHorizontal, "Start",
				40, gtk.ButtonBoxStyleStart), true, true, 5)
		vbox.PackStart(
			bb.CreateBBox(gtk.OrientationHorizontal, "End",
				40, gtk.ButtonBoxStyleEnd), true, true, 5)


		frame_vert := gtk.NewFrame("Vertical Button Boxes")
		main_vbox.PackStart(frame_vert, true, true, 10)

		hbox := gtk.NewBox(gtk.OrientationHorizontal, 0)
		hbox.SetBorderWidth(10)
		frame_vert.Add(hbox)

		hbox.PackStart(
			bb.CreateBBox(gtk.OrientationVertical, "Spread",
				30, gtk.ButtonBoxStyleSpread), true, true, 0)
		hbox.PackStart(
			bb.CreateBBox(gtk.OrientationVertical, "Edge",
				30, gtk.ButtonBoxStyleEdge), true, true, 5)
		hbox.PackStart(
			bb.CreateBBox(gtk.OrientationVertical, "Start",
				30, gtk.ButtonBoxStyleStart), true, true, 5)
		hbox.PackStart(
			bb.CreateBBox(gtk.OrientationVertical, "End",
				30, gtk.ButtonBoxStyleEnd), true, true, 5)
	}

	if (!bb.window.GetVisible()) {
		bb.window.ShowAll()
	} else {
		bb.window.Destroy()
		bb.window = nil
	}
	return bb.window
}
