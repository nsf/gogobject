package main

import (
	"gobject/gobject-2.0"
	"gobject/gtk-3.0"
	"gobject/pango-1.0"
	"os"
)

const (
	TitleColumn = iota
	FilenameColumn
	AppColumn
	StyleColumn
)

func CreateTreeView() *gtk.TreeView {
	model := gtk.NewTreeStore(
		gobject.String,      // title
		gobject.String,      // filename
		gobject.GoInterface, // app
		gobject.Int,         // style
	)

	treeview := gtk.NewTreeViewWithModel(model)
	selection := treeview.GetSelection()
	selection.SetMode(gtk.SelectionModeBrowse)
	treeview.SetSizeRequest(200, -1)

	for _, demo := range demos {
		iter := model.Append(nil, demo.Title, demo.Filename, demo.App, pango.StyleNormal)
		for _, cdemo := range demo.Children {
			model.Append(&iter, cdemo.Title, demo.Filename, cdemo.App, pango.StyleNormal)
		}
	}

	r := gtk.NewCellRendererText()
	c := gtk.NewTreeViewColumnWithAttributes("Widget (double click for demo)", r,
		"text", TitleColumn,
		"style", StyleColumn)
	treeview.AppendColumn(c)

	iter, _ := model.GetIterFirst()
	selection.SelectIter(&iter)

	// TODO: selection.Connect("changed", ...)
	treeview.Connect("row-activated", func(treeview *gtk.TreeView, path *gtk.TreePath) {
		iter, _ := model.GetIter(path)
		var app interface{}
		var style pango.Style
		model.Get(&iter, AppColumn, &app, StyleColumn, &style)
		if style == pango.StyleItalic {
			style = pango.StyleNormal
		} else {
			style = pango.StyleItalic
		}

		model.Set(&iter, StyleColumn, style)
		w := app.(DemoApp).Do(gtk.ToWindow(treeview.GetToplevel()))
		if w != nil {
			// TODO
		}
	})
	treeview.CollapseAll()
	treeview.SetHeadersVisible(false)

	// HERE

	return treeview
}

func CreateText(is_source bool) (*gtk.ScrolledWindow, *gtk.TextBuffer) {
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

func NewNotebookPage(nb *gtk.Notebook, w gtk.WidgetLike, label string) {
	l := gtk.NewLabelWithMnemonic(label)
	nb.AppendPage(w, l)
}

func main() {
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
	treeview := CreateTreeView()
	hbox.PackStart(treeview, false, false, 0)

	// notebook
	notebook := gtk.NewNotebook()
	hbox.PackStart(notebook, true, true, 0)

	// info
	sw, infobuf := CreateText(false)
	NewNotebookPage(notebook, sw, "_Info")
	tag := gtk.NewTextTag("title")
	tag.SetProperty("font", "Sans 18")
	infobuf.GetTagTable().Add(tag)

	window.ShowAll()
	gtk.Main()
}
