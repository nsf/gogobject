package main

import "gobject/gtk-3.0"

type DemoApp interface {
	Do(mainwin *gtk.Window) *gtk.Window
}

type DemoDesc struct {
	Title string
	Filename string
	App DemoApp
	Children []*DemoDesc
}

var demos = []*DemoDesc{
	{Title: "Button Boxes", Filename: "button_boxes.go", App: &ButtonBoxes},
	{Title: "Links",        Filename: "links.go",        App: &Links},
}
