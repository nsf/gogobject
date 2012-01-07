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


PREREQ=$(patsubst %.go,%.$O,$(DEMOS))
TARG=gogtk-demo
GOFILES=\
	main.go \
	demos.go \


include $(GOROOT)/src/Make.cmd

%.$O: %.go
	$(GC) -o $*.$O $*.go
