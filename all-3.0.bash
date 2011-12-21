#!/bin/bash

error_exit() {
	echo -e "\033[1;31m!!!!!!!!! ERROR !!!!!!!!!!\033[0m"
	exit $?
}

build() {
	./go-gobject-gen -config config.json $1
	make -C $1 install || error_exit
}

./clean.bash

make -C gi install
make
build glib-2.0
build gobject-2.0
build atk-1.0
build cairo-1.0
build gio-2.0
build gdkpixbuf-2.0
build pango-1.0
build gdk-3.0
build gtk-3.0
build gtksource-3.0
