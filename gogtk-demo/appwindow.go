// Application main window
//
// Demonstrates a typical application window with menubar, toolbar, statusbar.
package appwindow

import "gobject/gtk-3.0"
import "fmt"

var window *gtk.Window
var infobar *gtk.InfoBar
var messagelabel *gtk.Label
var mgr *gtk.UIManager

func activate_action(action *gtk.Action) {
	name := action.GetName()
	typename := action.GetType().String()

	if name == "DarkTheme" {
		value := gtk.ToToggleAction(action).GetActive()
		settings := gtk.SettingsGetDefault()
		settings.SetProperty("gtk-application-prefer-dark-theme", value)
		return
	}

	dialog := gtk.NewMessageDialog(window, gtk.DialogFlagsDestroyWithParent,
		gtk.MessageTypeInfo, gtk.ButtonsTypeClose,
		`You activated action: "%s" of type "%s"`, name, typename)
	dialog.Connect("response", func() { dialog.Destroy() })
	dialog.Show()
}

func about_cb() {
	// TODO: implement gtk_show_about_dialog and better about dialog
	dialog := gtk.NewAboutDialog()
	dialog.SetName("GoGTK Demo")
	// TODO: change (C) to real copyright unicode symbol (need to fix syntax highlighting first)
	dialog.SetCopyright("(C) Copyright 201x nsf <no.smile.face@gmail.com>")
	dialog.SetWebsite("http://github.com/nsf/gogobject")
	dialog.Connect("response", func() { dialog.Destroy() })
	dialog.Show()
}

var entries = []gtk.ActionEntry{
	{ Name: "FileMenu",        Label: "_File" },
	{ Name: "OpenMenu",        Label: "_Open" },
	{ Name: "PreferencesMenu", Label: "_Preferences" },
	{ Name: "ColorMenu",       Label: "_Color" },
	{ Name: "ShapeMenu",       Label: "_Shape" },
	{ Name: "HelpMenu",        Label: "_Help" },

	{ "New",    gtk.StockNew,    "_New",        "<control>N", "Create a new file", activate_action },
	{ "Open",   gtk.StockOpen,   "_Open",       "<control>O", "Open a new file",   activate_action },
	{ "Save",   gtk.StockSave,   "_Save",       "<control>S", "Save current file", activate_action },
	{ "SaveAs", gtk.StockSave,   "Save _As...", "",           "Save to a file",    activate_action },
	{ "Quit",   gtk.StockQuit,   "_Quit",       "<control>Q", "Quit",              activate_action },
	{ "About",  "",              "_About",      "<control>A", "About",             about_cb },
	{ "Logo",   "demo-gtk-logo", "",            "",           "GTK+",              activate_action },
}

var toggle_entries = []gtk.ToggleActionEntry{
	{ "Bold",      gtk.StockBold, "_Bold",              "<control>B", "Bold",              activate_action, true },
	{ "DarkTheme", "",            "_Prefer Dark Theme", "",           "Prefer Dark Theme", activate_action, false },
}

const (
	color_red = iota
	color_green
	color_blue
)

var color_entries = []gtk.RadioActionEntry{
	{ "Red",   "", "_Red",   "<control>R", "Blood", color_red },
	{ "Green", "", "_Green", "<control>G", "Grass", color_green },
	{ "Blue",  "", "_Blue",  "<control>B", "Sky",   color_blue },
}

const (
	shape_square = iota
	shape_rectangle
	shape_oval
)

var shape_entries = []gtk.RadioActionEntry{
	{ "Square",    "", "_Square",    "<control>S", "Square",    shape_square },
	{ "Rectangle", "", "_Rectangle", "<control>R", "Rectangle", shape_rectangle },
	{ "Oval",      "", "_Oval",      "<control>O", "Egg",       shape_oval },
}

var ui_info = `
<ui>
  <menubar name='MenuBar'>
    <menu action='FileMenu'>
      <menuitem action='New'/>
      <menuitem action='Open'/>
      <menuitem action='Save'/>
      <menuitem action='SaveAs'/>
      <separator/>
      <menuitem action='Quit'/>
    </menu>
    <menu action='PreferencesMenu'>
      <menuitem action='DarkTheme'/>
      <menu action='ColorMenu'>
       <menuitem action='Red'/>
       <menuitem action='Green'/>
       <menuitem action='Blue'/>
      </menu>
      <menu action='ShapeMenu'>
        <menuitem action='Square'/>
        <menuitem action='Rectangle'/>
        <menuitem action='Oval'/>
      </menu>
      <menuitem action='Bold'/>
    </menu>
    <menu action='HelpMenu'>
      <menuitem action='About'/>
    </menu>
  </menubar>
  <toolbar name='ToolBar'>
    <toolitem action='Open'/>
    <toolitem action='Quit'/>
    <separator/>
    <toolitem action='Logo'/>
  </toolbar>
</ui>
`

var stock_icons_registered bool

func register_stock_icons() {
	if stock_icons_registered {
		return
	}

	// TODO: Implement StockItem and StockAdd (rename it to StockAddItems due to consts clash)
	stock_icons_registered = true
}

func activate_radio_action(action *gtk.Action, current *gtk.RadioAction) {
	name := current.GetName()
	typename := current.GetType().String()
	active := current.GetActive()
	value := current.GetCurrentValue()
	if active {
		text := fmt.Sprintf("You activated radio action: \"%s\" of type \"%s\".\n" +
			"Current value: %d", name, typename, value)
		messagelabel.SetText(text)
		infobar.SetMessageType(gtk.MessageType(value))
		infobar.Show()
	}
}

func Do(mainwin *gtk.Window) *gtk.Window {
	if window == nil {
		register_stock_icons()

		// Create the toplevel window
		window = gtk.NewWindow(gtk.WindowTypeToplevel)
		window.SetScreen(mainwin.GetScreen())
		window.SetTitle("Application Window")
		window.SetIconName("document-open")
		window.Connect("destroy", func() {
			window = nil
			infobar = nil
			messagelabel = nil
			mgr = nil
		})

		table := gtk.NewGrid()
		window.Add(table)

		// Create the menubar and toolbar
		action_group := gtk.NewActionGroup("AppWindowActions")
		action_group.AddActions(entries)
		action_group.AddToggleActions(toggle_entries)
		action_group.AddRadioActions(color_entries, color_red, activate_radio_action)
		action_group.AddRadioActions(shape_entries, shape_square, activate_radio_action)

		mgr = gtk.NewUIManager()
		mgr.InsertActionGroup(action_group, 0)
		window.AddAccelGroup(mgr.GetAccelGroup())

		_, err := mgr.AddUiFromString(ui_info, -1)
		if err != nil {
			println("building menus failed: ", err.Error())
		}

		bar := mgr.GetWidget("/MenuBar")
		bar.Show()
		bar.SetHalign(gtk.AlignFill)
		table.Attach(bar, 0, 0, 1, 1)

		bar = mgr.GetWidget("/ToolBar")
		bar.Show()
		bar.SetHalign(gtk.AlignFill)
		table.Attach(bar, 0, 1, 1, 1)

		// Create document
		infobar = gtk.NewInfoBar()
		infobar.SetNoShowAll(true)
		messagelabel = gtk.NewLabel("")
		messagelabel.Show()
		gtk.ToBox(infobar.GetContentArea()).PackStart(messagelabel, true, true, 0)
		infobar.AddButton(gtk.StockOk, gtk.ResponseTypeOk)
		infobar.Connect("response", func() { infobar.Hide() })

		infobar.SetHalign(gtk.AlignFill)
		table.Attach(infobar, 0, 2, 1, 1)

		sw := gtk.NewScrolledWindow(nil, nil)
		sw.SetPolicy(gtk.PolicyTypeAutomatic, gtk.PolicyTypeAutomatic)
		sw.SetShadowType(gtk.ShadowTypeIn)

		sw.SetHalign(gtk.AlignFill)
		sw.SetValign(gtk.AlignFill)
		sw.SetHExpand(true)
		sw.SetVExpand(true)
		table.Attach(sw, 0, 3, 1, 1)

		window.SetDefaultSize(200, 200)

		contents := gtk.NewTextView()
		contents.GrabFocus()
		sw.Add(contents)

		// Create statusbar
		statusbar := gtk.NewStatusbar()
		statusbar.SetHalign(gtk.AlignFill)
		table.Attach(statusbar, 0, 4, 1, 1)

		// Show text widget info in the statusbar
		buffer := contents.GetBuffer()
		update_statusbar := func() {
			statusbar.Pop(0)
			count := buffer.GetCharCount()
			iter := buffer.GetIterAtMark(buffer.GetInsert())
			row := iter.GetLine()
			col := iter.GetLineOffset()
			msg := fmt.Sprintf("Cursor at row %d column %d - %d chars in document",
				row, col, count)
			statusbar.Push(0, msg)
		}

		buffer.Connect("changed", update_statusbar)
		buffer.Connect("mark_set", update_statusbar)
		update_statusbar()

	}

	if !window.GetVisible() {
		window.ShowAll()
	} else {
		window.Destroy()
	}
	return window
}