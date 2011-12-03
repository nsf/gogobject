#!/bin/bash

build() {
    ./go-gobject-gen -config config.json $1
    make -C $1 install
}

./clean.bash

make -C gi install
make
build gobject-2.0
build cairo-1.0
build gio-2.0
build gdkpixbuf-2.0
build gdk-3.0
build pango-1.0
build gtk-3.0
