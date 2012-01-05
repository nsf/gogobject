// Color Selector
//
// GtkColorSelection lets the user choose a color. GtkColorSelectionDialog is
// a prebuilt dialog containing a GtkColorSelection.
package color_selector

import "gobject/gtk-3.0"
import "gobject/gdk-3.0"
import "gobject/cairo-1.0"

var color gdk.RGBA
var window *gtk.Window

func Do(mainwin *gtk.Window) *gtk.Window {
	if window == nil {
		color = gdk.RGBA{0, 0, 1, 1}
		window = gtk.NewWindow(gtk.WindowTypeToplevel)
		window.SetScreen(mainwin.GetScreen())
		window.SetTitle("Color Selection")
		window.Connect("destroy", func() { window = nil })

		window.SetBorderWidth(8)

		vbox := gtk.NewBox(gtk.OrientationVertical, 8)
		vbox.SetBorderWidth(8)
		window.Add(vbox)

		// Create the color swatch area

		frame := gtk.NewFrame("")
		frame.SetShadowType(gtk.ShadowTypeIn)
		vbox.PackStart(frame, true, true, 0)

		da := gtk.NewDrawingArea()
		da.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {
			gdk.CairoSetSourceRgba(cr, &color)
			cr.Paint()
		})

		// set a minimum size
		da.SetSizeRequest(200, 200)
		frame.Add(da)

		button := gtk.NewButtonWithMnemonic("_Change the above color")
		button.SetHalign(gtk.AlignEnd)
		button.SetValign(gtk.AlignCenter)

		vbox.PackStart(button, false, false, 0)
		button.Connect("clicked", func() {
			dialog := gtk.NewColorSelectionDialog("Changing color")
			dialog.SetTransientFor(window)
			colorsel := gtk.ToColorSelection(dialog.GetColorSelection())
			colorsel.SetPreviousRgba(&color)
			colorsel.SetCurrentRgba(&color)
			colorsel.SetHasPalette(true)
			response := dialog.Run()
			if response == gtk.ResponseTypeOk {
				color = colorsel.GetCurrentRgba()
				da.QueueDraw()
			}
			dialog.Destroy()
		})
	}

	if !window.GetVisible() {
		window.ShowAll()
	} else {
		window.Destroy()
	}
	return window
}
