include $(GOROOT)/src/Make.inc

DEMOS=\
	button_boxes.go \
	links.go \
	list_store.go \
	spinner.go \
	expander.go \
	color_selector.go \
	info_bar.go \
	entry_buffer.go \
	pickers.go \
	main_window.go \
	paned_widgets.go \
	builder.go \
	assistant.go \
	size_groups.go \
	stock_browser.go \
	menus.go \
	entry_completion.go \
	drawing_area.go \
	combo_boxes.go \
	dialog.go \
	search_entry.go \
	iconview_edit.go \


PREREQ=$(patsubst %.go,%.$O,$(DEMOS)) common.$O
TARG=gogtk-demo
GOFILES=\
	main.go \
	demos.go \


include $(GOROOT)/src/Make.cmd

%.$O: %.go common.$O
	$(GC) -o $*.$O $*.go

common.$O: common.go
	$(GC) -o common.$O common.go
