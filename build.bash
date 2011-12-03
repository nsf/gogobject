#!/bin/bash

./go-gobject-gen -config config.json $@
make -C $@ install
