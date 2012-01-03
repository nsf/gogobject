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
)

type DemoFunc func(mainwin *gtk.Window) *gtk.Window

type DemoDesc struct {
	Title string
	Filename string
	Func DemoFunc
	Children []*DemoDesc
}

var demos = []*DemoDesc{
	{Title: "Button Boxes",   Filename: "button_boxes.go",   Func: button_boxes.Do},
	{Title: "Links",          Filename: "links.go",          Func: links.Do},
	{Title: "List Store",     Filename: "list_store.go",     Func: list_store.Do},
	{Title: "Spinner",        Filename: "spinner.go",        Func: spinner.Do},
	{Title: "Expander",       Filename: "expander.go",       Func: expander.Do},
	{Title: "Color Selector", Filename: "color_selector.go", Func: color_selector.Do},
	{Title: "Info bar",       Filename: "info_bar.go",       Func: info_bar.Do},
}
