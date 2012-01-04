package main

import (
	"gobject/gobject-2.0"
	"gobject/gdk-3.0"
	"gobject/gtk-3.0"
	"gobject/gdkpixbuf-2.0"
	"gobject/pango-1.0"
	"os"
)

const (
	title_column = iota
	filename_column
	func_column
	style_column
)

var infobuf *gtk.TextBuffer
var sourcebuf *gtk.TextBuffer

func create_tree_view() *gtk.Widget {
	model := gtk.NewTreeStore(
		gobject.String,      // title
		gobject.String,      // filename
		gobject.GoInterface, // app
		gobject.Int,         // style
	)

	tree_view := gtk.NewTreeViewWithModel(model)
	selection := tree_view.GetSelection()
	selection.SetMode(gtk.SelectionModeBrowse)
	tree_view.SetSizeRequest(200, -1)

	for _, demo := range demos {
		iter := model.Append(nil, demo.Title, demo.Filename, demo.Func, pango.StyleNormal)
		for _, cdemo := range demo.Children {
			model.Append(&iter, cdemo.Title, demo.Filename, cdemo.Func, pango.StyleNormal)
		}
	}

	r := gtk.NewCellRendererText()
	c := gtk.NewTreeViewColumnWithAttributes("Widget (double click for demo)", r,
		"text", title_column,
		"style", style_column)
	tree_view.AppendColumn(c)

	iter, _ := model.GetIterFirst()
	selection.SelectIter(&iter)

	// TODO: selection.Connect("changed", ...)
	tree_view.Connect("row-activated", func(tree_view *gtk.TreeView, path *gtk.TreePath) {
		iter, _ := model.GetIter(path)
		var app interface{}
		var style pango.Style
		model.Get(&iter, func_column, &app, style_column, &style)
		if style == pango.StyleItalic {
			style = pango.StyleNormal
		} else {
			style = pango.StyleItalic
		}

		if app.(DemoFunc) == nil {
			return
		}

		model.Set(&iter, style_column, style)
		w := app.(DemoFunc)(gtk.ToWindow(tree_view.GetToplevel()))
		if w != nil {
			w.Connect("destroy", func() {
				var style pango.Style
				model.Get(&iter, style_column, &style)
				if style == pango.StyleItalic {
					model.Set(&iter, style_column, pango.StyleNormal)
				}
			})
		}
	})
	tree_view.CollapseAll()
	tree_view.SetHeadersVisible(false)

	scrolled_window := gtk.NewScrolledWindow(nil, nil)
	scrolled_window.SetPolicy(gtk.PolicyTypeNever, gtk.PolicyTypeAutomatic)
	scrolled_window.Add(tree_view)

	label := gtk.NewLabel("Widget (double click for demo)")

	nb := gtk.NewNotebook()
	nb.AppendPage(scrolled_window, label)
	tree_view.GrabFocus()

	return gtk.ToWidget(nb)
}

func create_text(is_source bool) (*gtk.ScrolledWindow, *gtk.TextBuffer) {
	sw := gtk.NewScrolledWindow(nil, nil)
	sw.SetPolicy(gtk.PolicyTypeAutomatic, gtk.PolicyTypeAutomatic)
	sw.SetShadowType(gtk.ShadowTypeIn)

	tv := gtk.NewTextView()
	sw.Add(tv)

	buf := gtk.NewTextBuffer(nil)
	tv.SetBuffer(buf)
	tv.SetEditable(false)
	tv.SetCursorVisible(false)
	if is_source {
		tv.SetWrapMode(gtk.WrapModeNone)
	} else {
		tv.SetWrapMode(gtk.WrapModeWord)
	}

	return sw, buf
}

func find_file(name string) string {
	// TODO: no heuristic yet
	return name
}

func setup_default_icon() {
	filename := find_file("gtk-logo-rgb.gif")
	pixbuf, err := gdkpixbuf.NewPixbufFromFile(filename)
	if err != nil {
		dialog := gtk.NewMessageDialog(nil, 0, gtk.MessageTypeError, gtk.ButtonsTypeClose,
			"Failed to read icon file: %s", err)
		dialog.Connect("response", func() { dialog.Destroy() })
		dialog.ShowAll()
	} else {
		pixbuf = pixbuf.AddAlpha(true, 0xFF, 0xFF, 0xFF)
		gtk.WindowSetDefaultIcon(pixbuf)
	}
}

func main() {
	gdk.ThreadsInit()
	gtk.Init(os.Args)
	setup_default_icon()
	window := gtk.NewWindow(gtk.WindowTypeToplevel)
	window.SetTitle("GoGTK Code Demos")
	window.Connect("destroy", func() {
		gtk.MainQuit()
	})
	window.SetDefaultSize(800, 400)

	hbox := gtk.NewHBox(false, 3)
	window.Add(hbox)

	// treeview
	tree_view := create_tree_view()
	hbox.PackStart(tree_view, false, false, 0)

	// notebook
	notebook := gtk.NewNotebook()
	hbox.PackStart(notebook, true, true, 0)

	// info
	var sw *gtk.ScrolledWindow
	sw, infobuf = create_text(false)
	notebook.AppendPage(sw, gtk.NewLabelWithMnemonic("_Info"))
	infobuf.CreateTag("title",
		"font", "Sans 18")

	// source
	sw, sourcebuf = create_text(true)
	notebook.AppendPage(sw, gtk.NewLabelWithMnemonic("_Source"))
	sourcebuf.CreateTag("comment",
		"foreground", "DodgerBlue")
	sourcebuf.CreateTag("type",
		"foreground", "ForestGreen")
	sourcebuf.CreateTag("string",
		"foreground", "RosyBrown",
		"weight", pango.WeightBold)
	sourcebuf.CreateTag("control",
		"foreground", "purple")
	sourcebuf.CreateTag("function",
		"weight", pango.WeightBold,
		"foreground", "DarkGoldenrod4")

	window.ShowAll()
	gdk.ThreadsEnter()
	gtk.Main()
	gdk.ThreadsLeave()
}
