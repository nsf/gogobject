#!/bin/bash

error_exit() {
	echo -e "\033[1;31m!!!!!!!!! ERROR !!!!!!!!!!\033[0m"
	exit $?
}

build() {
	./gogobject -config config.json $1
}

#./clean.bash

go build || error_exit

for pkg in glib-2.0 gobject-2.0 cairo-1.0 atk-1.0 gio-2.0 gdkpixbuf-2.0 pango-1.0 pangocairo-1.0 gdk-2.0 gtk-2.0
do
	echo  "Installing $pkg"
	go install "./${pkg}" || error_exit
done
