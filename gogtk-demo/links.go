package main

import "gobject/gtk-3.0"
import "os"

const labelText =
`Some <a href="http://en.wikipedia.org/wiki/Text" title="plain text">text</a> may be marked up
as hyperlinks, which can be clicked
or activated via <a href="keynav">keynav</a>`

const dialogText =
`The term <i>keynav</i> is a shorthand for ` +
`keyboard navigation and refers to the process of using ` +
`a program (exclusively) via keyboard input.`

func ActivateLink(label *gtk.Label, uri string) bool {
	if uri == "keynav" {
		parent, ok := gtk.ToWindow(label.GetParent())
		if !ok {
			panic("bad type")
		}
		dialog := gtk.NewMessageDialogWithMarkup(parent,
			gtk.DialogFlagsDestroyWithParent,
			gtk.MessageTypeInfo,
			gtk.ButtonsTypeOk,
			dialogText)
		dialog.Present()
		dialog.Connect("response", func() {
			dialog.Destroy()
		})

		return true
	}

	return false
}


func Links() *gtk.Window {
	window := gtk.NewWindow(gtk.WindowTypeToplevel)
	window.SetTitle("Links")
	window.SetBorderWidth(12)

	label := gtk.NewLabel(labelText)
	label.SetUseMarkup(true)
	label.Connect("activate-link", ActivateLink)
	window.Add(label)
	window.ShowAll()
	return window
}

func main() {
	gtk.Init(os.Args)
	Links()
	gtk.Main()
}
