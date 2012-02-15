#/bin/bash

for pkg in glib-2.0 gobject-2.0 cairo-1.0 atk-1.0 gio-2.0 gdkpixbuf-2.0 pango-1.0 pangocairo-1.0 gdk-2.0 gtk-2.0 gdk-3.0 gtk-3.0 gtksource-3.0
do
	./gogobject -config config.json $pkg
done
