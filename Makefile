include $(GOROOT)/src/Make.inc

TARG=go-gobject-gen
GOFILES=binding_generator.go \
	util.go \
	function_builder.go \
	main.go \
	type.go \
	typeconv.go \
	templates.go \
	comment_skipper.go \
	cairo.go

include $(GOROOT)/src/Make.cmd
