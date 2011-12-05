#!/bin/bash

clean_in_dir() {
    make -C $1 clean
}

make clean
clean_in_dir gi
clean_in_dir gobject-2.0
clean_in_dir cairo-1.0
clean_in_dir gio-2.0
clean_in_dir gdkpixbuf-2.0
clean_in_dir gdk-3.0
clean_in_dir pango-1.0
clean_in_dir gtk-3.0
clean_in_dir gtksource-3.0

rm -f gobject-2.0/gobject.go
rm -f cairo-1.0/cairo.go
rm -f gio-2.0/gio.go
rm -f gdkpixbuf-2.0/gdkpixbuf.go
rm -f gdk-3.0/gdk.go
rm -f pango-1.0/pango.go
rm -f gtk-3.0/gtk.go
rm -f gtksource-3.0/gtksource.go
