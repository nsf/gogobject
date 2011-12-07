package main

import "gobject/gtk-3.0"
import "os"

func Spinner() *gtk.Dialog {
	window := gtk.NewDialogWithButtons("GtkSpinner", nil, 0,
		gtk.StockClose, gtk.ResponseTypeNone)

	window.SetResizable(false)
	window.Connect("response", func() {
		window.Destroy()
	})

	content_area := gtk.ToBox(window.GetContentArea())
	vbox := gtk.NewBox(gtk.OrientationVertical, 5)
	content_area.PackStart(vbox, true, true, 0)
	vbox.SetBorderWidth(5)

	var hbox *gtk.Box

	// Sensitive
	hbox = gtk.NewBox(gtk.OrientationHorizontal, 5)
	spinner_sensitive := gtk.NewSpinner()
	hbox.Add(spinner_sensitive)
	hbox.Add(gtk.NewEntry())
	vbox.Add(hbox)

	// Disabled
	hbox = gtk.NewBox(gtk.OrientationHorizontal, 5)
	spinner_unsensitive := gtk.NewSpinner()
	hbox.Add(spinner_unsensitive)
	hbox.Add(gtk.NewEntry())
	vbox.Add(hbox)
	hbox.SetSensitive(false)

	button := gtk.NewButtonFromStock(gtk.StockMediaPlay)
	button.Connect("clicked", func() {
		spinner_sensitive.Start()
		spinner_unsensitive.Start()
	})
	vbox.Add(button)

	button = gtk.NewButtonFromStock(gtk.StockMediaStop)
	button.Connect("clicked", func() {
		spinner_sensitive.Stop()
		spinner_unsensitive.Stop()
	})
	vbox.Add(button)

	spinner_sensitive.Start()
	spinner_unsensitive.Start()

	window.ShowAll()
	return window
}

func main() {
	gtk.Init(os.Args)
	Spinner()
	gtk.Main()
}
