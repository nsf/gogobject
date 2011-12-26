package main

import "gobject/gtk-3.0"

type LinksApp struct {
	window *gtk.Window
}

var Links LinksApp

const links_labelText =
`Some <a href="http://en.wikipedia.org/wiki/Text" title="plain text">text</a> may be marked up
as hyperlinks, which can be clicked
or activated via <a href="keynav">keynav</a>`

const links_dialogText =
`The term <i>keynav</i> is a shorthand for ` +
`keyboard navigation and refers to the process of using ` +
`a program (exclusively) via keyboard input.`


func (this *LinksApp) Do(mainwin *gtk.Window) *gtk.Window {
	if this.window == nil {
		this.window = gtk.NewWindow(gtk.WindowTypeToplevel)
		this.window.SetTitle("Links")
		this.window.SetBorderWidth(12)

		label := gtk.NewLabel(links_labelText)
		label.SetUseMarkup(true)
		label.Connect("activate-link", links_ActivateLink)
		this.window.Add(label)
	}

	if !this.window.GetVisible() {
		this.window.ShowAll()
	} else {
		this.window.Destroy()
		this.window = nil
	}
	return this.window
}

func links_ActivateLink(label *gtk.Label, uri string) bool {
	if uri == "keynav" {
		parent := gtk.ToWindow(label.GetParent())
		dialog := gtk.NewMessageDialogWithMarkup(parent,
			gtk.DialogFlagsDestroyWithParent,
			gtk.MessageTypeInfo,
			gtk.ButtonsTypeOk,
			links_dialogText)
		dialog.Present()
		dialog.Connect("response", func() {
			dialog.Destroy()
		})

		return true
	}

	return false
}
