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
	# build cairo-1.0, it doesn't depend on bindings generator
	bld(
		features='cgo go gopackage',
		cgo_source='cairo-1.0/cairo.go',
		source='cairo-1.0/types_%s.go cairo-1.0/cairo.c' % bld.env.GOARCH,
		target='gobject/cairo-1.0',
		uselib='CAIRO',
	)

	# bindings generator
	bld(
		features='cgo go gopackage',
		cgo_source='gi/gi.go',
		target='gobject/gi',
		uselib='GI',
	)
	gggtg = bld(
		features='go goprogram',
		source=bld.path.ant_glob('*.go'),
		target='go-gobject-gen',
		install_path=None, # don't install
	)
	gggtg.post()

	# now the set of generated library bindings
	cfg = bld.path.find_resource('config.json')
	bld.env.GGGFLAGS = ['-config=' + cfg.bldpath()]

	libdirs = [
		"atk-1.0",
		"gdk-3.0",
		"gdkpixbuf-2.0",
		"gio-2.0",
		"glib-2.0",
		"gobject-2.0",
		"gtk-3.0",
		"gtksource-3.0",
		"pango-1.0",
		"pangocairo-1.0",
	]

	for dir in [bld.path.find_dir(x) for x in libdirs]:
		# gogobject tasks are created by extension mechanism
		tg = bld(source=dir.ant_glob('*.go.in'))
		tg.post()

		# get generated source nodes
		gen_go = tg.tasks[0].outputs[0]
		gen_c = tg.tasks[0].outputs[1]

		# the bindings library
		bld(
			features='cgo go gopackage',
			source=dir.get_src().ant_glob('*.c') + [gen_c],
			cgo_source=gen_go,
			target='gobject/' + dir.name,
			uselib=dir.name[:-4].upper(),
			includes=dir.name,
		)

	# and finally the demo app
	bld.recurse('gogtk-demo')
