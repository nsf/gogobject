# -*- mode: python -*-

top = '.'
out = 'build'

def configure(conf):
	conf.load('gcc')
	conf.load('go cgo gogobject', tooldir='waftools')

	def first_word(s):
		space = s.find(' ')
		if space == -1:
			return s
		return s[:space]

	def check_cfg(package, uselib_store):
		conf.check_cfg(
			package=package,
			args=['--cflags', '--libs'],
			uselib_store=uselib_store,
			msg="Checking for '%s'" % first_word(package),
		)

	check_cfg('gobject-introspection-1.0', 'GI')
	check_cfg('cairo cairo-png cairo-pdf cairo-gobject', 'CAIRO')
	check_cfg('atk', 'ATK')
	check_cfg('gdk-3.0', 'GDK')
	check_cfg('gdk-pixbuf-2.0', 'GDKPIXBUF')
	check_cfg('gio-2.0', 'GIO')
	check_cfg('glib-2.0', 'GLIB')
	check_cfg('gobject-2.0', 'GOBJECT')
	check_cfg('gtk+-3.0', 'GTK')
	check_cfg('gtksourceview-3.0', 'GTKSOURCE')
	check_cfg('pango', 'PANGO')
	check_cfg('pangocairo', 'PANGOCAIRO')

	conf.env.append_value('INCLUDES', conf.bldnode.abspath())

def build(bld):
	def cgopackage(source, target, uselib):
		bld(
			features='cgo go gopackage',
			cgo_source=source,
			target=target,
			uselib=uselib,
		)

	cgopackage('gi/gi.go', 'gobject/gi', 'GI')
	gggtg = bld(
		features='go goprogram',
		source=bld.path.ant_glob('*.go'),
		target='go-gobject-gen',
	)
	gggtg.post()

	bld.add_group()

	# build cairo-1.0, it doesn't depend on bindings generator
	bld(
		features='cgo go gopackage',
		cgo_source='cairo-1.0/cairo.go',
		source='cairo-1.0/types_%s.go cairo-1.0/cairo.c' % bld.env.GOARCH,
		target='gobject/cairo-1.0',
		uselib='CAIRO',
	)

	# now the set of generated bindings
	cfg = bld.srcnode.find_node('config.json')
	bld.env.GGGFLAGS = ['-config=' + cfg.bldpath()]

	for dir in bld.path.ant_glob('*-?.?', excl=['gdk-2.0', 'gtk-2.0', 'cairo-1.0'], dir=True):
		# generator, automatically hooked up by extension
		tg = bld(
			source=dir.ant_glob('*.go.in')
		)
		tg.post()

		# the bindings library itself
		bld(
			features='cgo go gopackage',
			source=dir.get_src().ant_glob('*.c'),
			cgo_source=tg.tasks[0].outputs[0],
			target='gobject/' + dir.name,
			uselib=dir.name[:-4].upper(),
			includes=[dir],
		)

	# and finally the demo app
	for demoapp in bld.path.ant_glob('gogtk-demo/*.go', excl=['gogtk-demo/demos.go', 'gogtk-demo/main.go']):
		bld(
			features='go',
			source=demoapp,
			target='gogtk-demo/' + demoapp.name[:-3],
		)

	bld(
		features='go goprogram',
		source='gogtk-demo/main.go gogtk-demo/demos.go',
		target='gogtk-demo/gogtk-demo',
	)
