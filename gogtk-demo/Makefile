include $(GOROOT)/src/Make.inc

DEMOS=\
	button_boxes.go \
	links.go \
	list_store.go \
	spinner.go \


PREREQ=$(patsubst %.go,%.$O,$(DEMOS))
TARG=gogtk-demo
GOFILES=\
	main.go \
	demos.go \


include $(GOROOT)/src/Make.cmd

%.$O: %.go
	$(GC) -o $*.$O $*.go
