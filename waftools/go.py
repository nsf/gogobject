################################################################################
# Possible scenarios (@ - many, % - one):
#
# 1. go object files:
#    @.go -> %.6
#
#    One object file per Go package.
#
# 2. go packages
#    %.6 -> %.a
#
#    Technically a Go package is an object file, but if it's a cgo case or there
#    are C/asm extensions, .a file may contain more than one object file.
#
# 3. go programs
#    %.6 -> % (or %.exe on windows)
#
#    Package should be called 'main' and there should be 'main.main' function.
#
# 4. cgo packages
#     cgo | @.go -> _cgo_defun.c, _cgo_export.c, _cgo_export.h, _cgo_gotypes.go, _cgo_main.c, @.cgo1.go, @.cgo2.c
#      6c | _cgo_defun.c -> _cgo_defun.6
#     gcc | _cgo_main.c, _cgo_export.c, @.cgo2.c -> _cgo_main.o, _cgo_export.o, @.cgo2.o
#     gcc | _cgo_main.o, _cgo_export.o, @.cgo2.o -> _cgo_.o
#     cgo | _cgo_.o -> _cgo_import.c
#      6c | _cgo_import.c -> _cgo_import.6
#      6g | _cgo_gotypes.go, @.cgo1.go -> _go_.6
#    pack | _go_.6, _cgo_import.6, _cgo_defun.6, _cgo_export.o, @.cgo2.o -> %.a
#
#    It's a bit complicated, yeah. But we have to handle that.
#
################################################################################
# Misc notes:
#
# 1. Go linker links libraries automatically, it's not like C where you have to
#    add a bunch of -lmylibX.
################################################################################
# TODO:
# 1. Add support for this (from ccroot's create_compiled_task):
#    out = '%s.%d.o' % (node.name, self.idx)
################################################################################

import os, platform, sys, tempfile, go_scan
from waflib import Utils
from waflib.Configure import conf
from waflib.Utils import subprocess
from waflib.Task import Task
from waflib.TaskGen import feature, before_method, after_method, taskgen_method
from waflib.Tools.ccroot import stlink_task, link_task, apply_incpaths, apply_link
from waflib.Errors import WafError

########################################################################
# Tasks and task generators
########################################################################

class go(Task):
	run_str = '${GO_6G} ${GCFLAGS} ${GCPATH_ST:INCPATHS} -o ${TGT} ${SRC}'
	scan = go_scan.scan

class gopackage(stlink_task):
	run_str = '${GO_PACK} grc ${TGT} ${SRC}'
	inst_to = os.path.join('${GOROOT}', 'pkg', '${GOOS}_${GOARCH}')

class goprogram(link_task):
	run_str = '${GO_6L} ${GLFLAGS} ${GLPATH_ST:INCPATHS} -o ${TGT} ${SRC}'
	inst_to = '${BINDIR}'
	chmod   = Utils.O755

@taskgen_method
def extract_nodes_with_ext(self, fromattr, ext):
	src_nodes = []
	no_nodes = []
	for n in self.to_nodes(getattr(self, fromattr)):
		if n.name.endswith(ext):
			src_nodes.append(n)
		else:
			no_nodes.append(n)
	setattr(self, fromattr, no_nodes)
	return src_nodes

@feature('goprogram')
@before_method('apply_link')
def apply_cgo_link(self):
	# special case, when user wants a 'goprogram' with 'cgo', we need an
	# intermediate 'gopackage' to be inserted, because go linker links only
	# one file at a time
	if 'cgo' not in self.features:
		return

	objs = [t.outputs[0] for t in getattr(self, 'compiled_tasks', [])]
	lt = self.create_task('gopackage', objs)
	lt.add_target(self.target)
	self.compiled_tasks = [lt]

@feature('go')
@before_method('process_source')
def apply_go(self):
	# extract .go nodes from 'source'
	src_nodes = self.extract_nodes_with_ext('source', '.go')

	# if there were no source nodes, return
	if not src_nodes:
		return

	# object file
	obj_node = self.path.find_or_declare(self.target + '.%s' % self.env.GOCHAR)
	task = self.create_task('go', src_nodes, obj_node)
	try:
		self.compiled_tasks.append(task)
	except AttributeError:
		self.compiled_tasks = [task]

def decorate(*args):
	decorators = args[:-1]
	func = args[-1]
	for d in reversed(decorators):
		func = d(func)

decorate(
	feature('go'),
	after_method('propagate_uselib_vars', 'process_source'),
	apply_incpaths,
)
decorate(
	feature('goprogram', 'gopackage'),
	after_method('process_source'),
	apply_link,
)

########################################################################
# Configuration
########################################################################

@conf
def find_go_command(self):
	# find go command, we will use it to run go tools
	self.find_program('go', var='GO')

@conf
def get_go_env(self):
	def set_def(var, val):
		if not self.env[var]:
			self.env[var] = val

	vars = {}
	try:
		out = self.cmd_and_log([self.env.GO, 'tool', 'dist', 'env'])
		for line in out.splitlines():
			eq = line.index('=')
			vars[line[:eq]] = line[eq+2:-1]
	except (WafError, ValueError):
		pass

	vars_to_grab = 'GOROOT GOBIN GOARCH GOOS GOHOSTARCH GOHOSTOS GOTOOLDIR GOCHAR'.split()
	for v in vars_to_grab:
		self.start_msg('Checking for %s' % v)
		if v not in vars:
			self.end_msg('no', color='YELLOW')
		else:
			set_def(v, vars[v])
			self.end_msg(self.env[v])

	# pattern for gopackage task output
	set_def('gopackage_PATTERN', '%s.a')

	# misc flags/patterns
	set_def('GCPATH_ST', '-I%s')
	set_def('GLPATH_ST', '-L%s')


@conf
def find_go_tools(self):
	def find_program(prog, var):
		self.find_program(prog, var=var, path_list=[self.env.GOTOOLDIR])

	c = self.env.GOCHAR
	find_program(c + 'c', 'GO_6C')
	find_program(c + 'g', 'GO_6G')
	find_program(c + 'l', 'GO_6L')
	find_program('pack',  'GO_PACK')
	find_program('cgo',   'GO_CGO')
	find_program('dist',  'GO_DIST')

@conf
def get_go_version(self):
	try:
		self.start_msg('Checking for go version')
		if not self.env.GOVERSION:
			version = self.cmd_and_log([self.env.GO_DIST, 'version']).strip()
			self.env.GOVERSION = version
		self.end_msg(self.env.GOVERSION)
	except WafError:
		self.end_msg('no', color="YELLOW")

def configure(conf):
	conf.find_go_command()
	conf.get_go_env()
	conf.find_go_tools()
	conf.get_go_version()

