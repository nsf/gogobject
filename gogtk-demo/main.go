package main

import (
	"gobject/gobject-2.0"
	"gobject/gdk-3.0"
	"gobject/gtk-3.0"
	"gobject/pango-1.0"
	"os"
)

const (
	title_column = iota
	filename_column
	func_column
	style_column
)

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

func new_notebook_page(nb *gtk.Notebook, w gtk.WidgetLike, label string) {
	l := gtk.NewLabelWithMnemonic(label)
	nb.AppendPage(w, l)
}

func main() {
	gdk.ThreadsInit()
	gtk.Init(os.Args)
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
	sw, infobuf := create_text(false)
	new_notebook_page(notebook, sw, "_Info")
	tag := gtk.NewTextTag("title")
	tag.SetProperty("font", "Sans 18")
	infobuf.GetTagTable().Add(tag)

	println(infobuf.GetTagTable())
	println(infobuf.GetTagTable())
	println(infobuf.GetTagTable())

	window.ShowAll()
	gdk.ThreadsEnter()
	gtk.Main()
	gdk.ThreadsLeave()
}
