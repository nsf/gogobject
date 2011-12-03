package main

import "gobject/gtk-3.0"
import "os"

func CreateBBox(orient gtk.Orientation, title string, spacing int, layout gtk.ButtonBoxStyle) *gtk.Frame {
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

func ButtonBoxes() *gtk.Window {
	window := gtk.NewWindow(gtk.WindowTypeToplevel)
	window.SetTitle("Button Boxes")
	window.SetBorderWidth(10)

	main_vbox := gtk.NewBox(gtk.OrientationVertical, 0)
	window.Add(main_vbox)

	frame_horz := gtk.NewFrame("Horizontal Button Boxes")
	main_vbox.PackStart(frame_horz, true, true, 10)

	vbox := gtk.NewBox(gtk.OrientationVertical, 0)
	vbox.SetBorderWidth(10)
	frame_horz.Add(vbox)

	vbox.PackStart(
		CreateBBox(gtk.OrientationHorizontal, "Spread",
			40, gtk.ButtonBoxStyleSpread), true, true, 0)
	vbox.PackStart(
		CreateBBox(gtk.OrientationHorizontal, "Edge",
			40, gtk.ButtonBoxStyleEdge), true, true, 5)
	vbox.PackStart(
		CreateBBox(gtk.OrientationHorizontal, "Start",
			40, gtk.ButtonBoxStyleStart), true, true, 5)
	vbox.PackStart(
		CreateBBox(gtk.OrientationHorizontal, "End",
			40, gtk.ButtonBoxStyleEnd), true, true, 5)

	frame_vert := gtk.NewFrame("Vertical Button Boxes")
	main_vbox.PackStart(frame_vert, true, true, 10)

	hbox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	hbox.SetBorderWidth(10)
	frame_vert.Add(hbox)

	hbox.PackStart(
		CreateBBox(gtk.OrientationVertical, "Spread",
			30, gtk.ButtonBoxStyleSpread), true, true, 0)
	hbox.PackStart(
		CreateBBox(gtk.OrientationVertical, "Edge",
			30, gtk.ButtonBoxStyleEdge), true, true, 5)
	hbox.PackStart(
		CreateBBox(gtk.OrientationVertical, "Start",
			30, gtk.ButtonBoxStyleStart), true, true, 5)
	hbox.PackStart(
		CreateBBox(gtk.OrientationVertical, "End",
			30, gtk.ButtonBoxStyleEnd), true, true, 5)

	window.ShowAll()
	return window
}

func main() {
	gtk.Init(os.Args)
	ButtonBoxes()
	gtk.Main()
}
