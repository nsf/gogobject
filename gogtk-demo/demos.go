package main

import "gobject/gtk-3.0"
import (
	"./button_boxes"
	"./links"
	"./list_store"
	"./spinner"
	"./expander"
	"./color_selector"
	"./info_bar"
	"./entry_buffer"
	"./pickers"
	"./main_window"
	"./paned_widgets"
	"./builder"
	"./assistant"
)

type DemoFunc func(mainwin *gtk.Window) *gtk.Window

type DemoDesc struct {
	Title string
	Filename string
	Func DemoFunc
	Children []*DemoDesc
}

var demos = []*DemoDesc{
	{Title: "Application main window", Filename: "main_window.go",    Func: main_window.Do},
	{Title: "Assistant",               Filename: "assistant.go",      Func: assistant.Do},
	{Title: "Builder",                 Filename: "builder.go",        Func: builder.Do},
	{Title: "Button Boxes",            Filename: "button_boxes.go",   Func: button_boxes.Do},
	{Title: "Color Selector",          Filename: "color_selector.go", Func: color_selector.Do},
	{Title: "Entry", Children: []*DemoDesc{
		{Title: "Entry Buffer", Filename: "entry_buffer.go", Func: entry_buffer.Do},
	}},
	{Title: "Expander",                Filename: "expander.go",       Func: expander.Do},
	{Title: "Info bar",                Filename: "info_bar.go",       Func: info_bar.Do},
	{Title: "Links",                   Filename: "links.go",          Func: links.Do},
	{Title: "Paned Widgets",           Filename: "paned_widgets.go",  Func: paned_widgets.Do},
	{Title: "Pickers",                 Filename: "pickers.go",        Func: pickers.Do},
	{Title: "Spinner",                 Filename: "spinner.go",        Func: spinner.Do},
	{Title: "Tree View", Children: []*DemoDesc{
		{Title: "List Store", Filename: "list_store.go", Func: list_store.Do},
	}},
}
