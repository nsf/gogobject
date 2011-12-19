// TODO: spinner stuff

package main

import (
	"gobject/gobject-2.0"
	"gobject/gtk-3.0"
	"os"
)

type Bug struct {
	fixed       bool
	number      int
	severity    string
	description string
}

const (
	ColumnFixed = iota
	ColumnNumber
	ColumnSeverity
	ColumnDescription
	ColumnPulse
	ColumnIcon
	ColumnActive
	ColumnSensitive
)

var data = []Bug{
	{false, 60482, "Normal", "scrollable notebooks and hidden tabs"},
	{false, 60620, "Critical", "gdk_window_clear_area (gdkwindow-win32.c) is not thread-safe"},
	{false, 50214, "Major", "Xft support does not clean up correctly"},
	{true, 52877, "Major", "GtkFileSelection needs a refresh method. "},
	{false, 56070, "Normal", "Can't click button after setting in sensitive"},
	{true, 56355, "Normal", "GtkLabel - Not all changes propagate correctly"},
	{false, 50055, "Normal", "Rework width/height computations for TreeView"},
	{false, 58278, "Normal", "gtk_dialog_set_response_sensitive () doesn't work"},
	{false, 55767, "Normal", "Getters for all setters"},
	{false, 56925, "Normal", "Gtkcalender size"},
	{false, 56221, "Normal", "Selectable label needs right-click copy menu"},
	{true, 50939, "Normal", "Add shift clicking to GtkTextView"},
	{false, 6112, "Enhancement", "netscape-like collapsable toolbars"},
	{false, 1, "Normal", "First bug :=)"},
}

func CreateModel() *gtk.ListStore {
	store := gtk.NewListStore(gobject.Boolean,
		gobject.Int,
		gobject.String,
		gobject.String,
		gobject.Int,
		gobject.String,
		gobject.Boolean,
		gobject.Boolean)

	for i := range data {
		var icon_name string
		var sensitive = true

		if i == 1 || i == 3 {
			icon_name = "battery-caution-charging-symbolic"
		}

		if i == 3 {
			sensitive = false
		}

		store.Append(data[i].fixed,
			data[i].number,
			data[i].severity,
			data[i].description,
			0,
			icon_name,
			false,
			sensitive)
	}

	return store
}

func AddColumns(treeview *gtk.TreeView, model *gtk.ListStore) {
	var r gtk.CellRendererLike
	var c *gtk.TreeViewColumn

	// column for fixed toggles
	toggle := gtk.NewCellRendererToggle()
	toggle.Connect("toggled", func(cell *gtk.CellRendererToggle, path_str string) {
		path := gtk.NewTreePathFromString(path_str)
		iter, _ := model.GetIter(path)
		v := model.GetValue(&iter, ColumnFixed)
		v.SetBool(!v.GetBool())
		model.SetValue(&iter, ColumnFixed, &v)
	})
	c = gtk.NewTreeViewColumn()
	c.SetTitle("Fixed?")
	c.PackStart(toggle, true)
	c.AddAttribute(toggle, "active", ColumnFixed)

	// set this column to a fixed sizing (of 50 pixels)
	c.SetSizing(gtk.TreeViewColumnSizingFixed)
	c.SetFixedWidth(50)
	treeview.AppendColumn(c)

	// column for bug numbers
	r = gtk.NewCellRendererText()
	c = gtk.NewTreeViewColumn()
	c.SetTitle("Bug number")
	c.PackStart(r, true)
	c.AddAttribute(r, "text", ColumnNumber)
	c.SetSortColumnId(ColumnNumber)
	treeview.AppendColumn(c)

	// column for severities
	r = gtk.NewCellRendererText()
	c = gtk.NewTreeViewColumn()
	c.SetTitle("Severity")
	c.PackStart(r, true)
	c.AddAttribute(r, "text", ColumnSeverity)
	c.SetSortColumnId(ColumnSeverity)
	treeview.AppendColumn(c)

	// column for description
	r = gtk.NewCellRendererText()
	c = gtk.NewTreeViewColumn()
	c.SetTitle("Description")
	c.PackStart(r, true)
	c.AddAttribute(r, "text", ColumnDescription)
	c.SetSortColumnId(ColumnDescription)
	treeview.AppendColumn(c)

	// column for spinner
	r = gtk.NewCellRendererSpinner()
	c = gtk.NewTreeViewColumn()
	c.SetTitle("Spinning")
	c.PackStart(r, true)
	c.AddAttribute(r, "pulse", ColumnPulse)
	c.AddAttribute(r, "active", ColumnActive)
	c.SetSortColumnId(ColumnPulse)
	treeview.AppendColumn(c)

	// column for symbolic icon
	r = gtk.NewCellRendererPixbuf()
	// TODO: r.Set("follow-state", true)
	c = gtk.NewTreeViewColumn()
	c.SetTitle("Symbolic icon")
	c.PackStart(r, true)
	c.AddAttribute(r, "icon-name", ColumnIcon)
	c.AddAttribute(r, "sensitive", ColumnSensitive)
	c.SetSortColumnId(ColumnIcon)
	treeview.AppendColumn(c)
}

func ListStore() *gtk.Window {
	window := gtk.NewWindow(gtk.WindowTypeToplevel)
	window.SetTitle("gtk.ListStore demo")
	window.SetBorderWidth(8)

	vbox := gtk.NewBox(gtk.OrientationVertical, 8)
	window.Add(vbox)

	label := gtk.NewLabel("This is the bug list (note: not based on real data, it would be nice to have a nice ODBC interface to bugzilla or so, though).")
	vbox.PackStart(label, false, false, 0)

	sw := gtk.NewScrolledWindow(nil, nil)
	sw.SetShadowType(gtk.ShadowTypeEtchedIn)
	sw.SetPolicy(gtk.PolicyTypeNever, gtk.PolicyTypeAutomatic)
	vbox.PackStart(sw, true, true, 0)

	// create tree model
	model := CreateModel()

	// create tree view
	treeview := gtk.NewTreeViewWithModel(model)
	treeview.SetRulesHint(true)
	treeview.SetSearchColumn(ColumnDescription)
	sw.Add(treeview)

	// add column to the tree view
	AddColumns(treeview, model)

	// finish & show
	window.SetDefaultSize(280, 250)
	window.ShowAll()
	return window
}

func main() {
	gtk.Init(os.Args)
	ListStore()
	gtk.Main()
}
