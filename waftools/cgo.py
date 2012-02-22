import os
from waflib.Task import Task
from waflib.TaskGen import feature, before_method, after_method
from waflib.Tools.ccroot import propagate_uselib_vars, process_use, USELIB_VARS
from go import decorate

USELIB_VARS['cgo'] = USELIB_VARS['c'] | USELIB_VARS['cshlib']

class cgo(Task):
	run_str = '${GO_CGO} ${CGOFLAGS} -- ${CFLAGS} ${CPPPATH_ST:INCPATHS} ${DEFINES_ST:DEFINES} ${SRC}'
	def __str__(self):
		# simple hack to display dynimport flag as a source, because
		# it's not an actual input even though in fact it is
		out = super(cgo, self).__str__()
		if hasattr(self, 'dynimport'):
			p1 = out[:5]
			p2 = out[5:]
			return p1 + self.dynimport.srcpath() + p2
		return out


# limited task for using plan9 C compiler (required for cgo)
class plan9c(Task):
	run_str = '${GO_6C} -FVw ${GCPATH_ST:INCPATHS} ${GOCFLAGS} -o ${TGT} ${SRC}'

@feature('cgo')
@before_method('apply_go')
def apply_cgo(self):
	if 'go' not in self.features:
		self.bld.fatal('"cgo" feature doesn\'t make sense without "go" feature')

	# task generator could be created in some other group than the current one
	current_group = self.bld.current_group
	self.bld.set_group(self.bld.get_group_idx(self))

	# extract .go nodes from 'cgo_source'
	src_nodes = self.extract_nodes_with_ext('cgo_source', '.go')
	tgt_nodes = []

	# prepare self.source for appending additional nodes
	self.source = self.to_nodes(self.source)

	# build directory
	bld_dir = self.path.get_bld()

	# 1. cgo tool, first run
	# @.go -> _cgo_defun.c, _cgo_export.c, _cgo_export.h,
	#         _cgo_gotypes.go, _cgo_main.c, @.cgo1.go, @.cgo2.c
	obj_dir = bld_dir.make_node('_cgoobj.%d' % self.idx)
	obj_dir.mkdir()
	self.env.append_value('INCLUDES', obj_dir.abspath())

	cgo_defun = obj_dir.make_node('_cgo_defun.c')
	cgo_export_c = obj_dir.make_node('_cgo_export.c')
	cgo_export_h = obj_dir.make_node('_cgo_export.h')
	cgo_gotypes = obj_dir.make_node('_cgo_gotypes.go')
	cgo_main = obj_dir.make_node('_cgo_main.c')

	tgt_nodes.extend([cgo_defun, cgo_export_c, cgo_export_h, cgo_gotypes, cgo_main])

	cgo1_nodes = []
	cgo2_nodes = []
	for node in src_nodes:
		out = node.bldpath().replace(os.sep, '_')[:-3]
		cgo1 = obj_dir.make_node(out + '.cgo1.go')
		cgo2 = obj_dir.make_node(out + '.cgo2.c')
		tgt_nodes.extend([cgo1, cgo2])
		cgo1_nodes.append(cgo1)
		cgo2_nodes.append(cgo2)

	# append necessary nodes to self.source, they will be handled by 'go' feature
	self.source.append(cgo_gotypes)
	for cgo1 in cgo1_nodes:
		self.source.append(cgo1)

	task = self.create_task('cgo', src_nodes, tgt_nodes)
	task.env.append_value('CGOFLAGS', '-objdir=' + obj_dir.bldpath())

	# 2. 6c defun
	# _cgo_defun.c -> _cgo_defun.6
	def create_plan9c_task(input):
		task = self.create_task('plan9c', input,
			input.change_ext('.' + self.env.GOCHAR))
		task.env.GOCFLAGS = ['-DGOOS_' + self.env.GOOS, '-DGOARCH_' + self.env.GOARCH]
		task.env.INCPATHS = [
			os.path.join(self.env.GOROOT, 'pkg', '%s_%s' % (self.env.GOOS, self.env.GOARCH)),
			obj_dir.abspath(),
		]
		try:
			self.compiled_tasks.append(task)
		except AttributeError:
			self.compiled_tasks = [task]

	create_plan9c_task(cgo_defun)

	# 3. gcc sources and binary
	# _cgo_main.c, _cgo_export.c, @.cgo2.c -> _cgo_main.o, _cgo_export.o, @.cgo2.o
	# _cgo_main.o, _cgo_export.o, @.cgo2.o -> _cgo_.o

	self.env.cprogram_PATTERN = '%s'
	self.env.append_unique('CFLAGS', self.env.GOGCCFLAGS)
	gcc_src_nodes = [cgo_main, cgo_export_c]
	cgo_obj = obj_dir.find_or_declare('_cgo_.o')
	for cgo2 in cgo2_nodes:
		gcc_src_nodes.append(cgo2)
	tg = self.bld(
		features='c cprogram',
		source = gcc_src_nodes + self.extract_nodes_with_ext('source', '.c'),
		target = cgo_obj,
		env = self.env,
		install_path = None,
	)
	tg.post()

	# 4. cgo import
	# _cgo_.o -> _cgo_import.c
	cgo_import = obj_dir.make_node('_cgo_import.c')
	task = self.create_task('cgo', None, cgo_import)
	task.env.append_value('CGOFLAGS', [
		'-objdir=' + obj_dir.bldpath(),
		'-dynimport=' + cgo_obj.bldpath(),
		'-dynout=' + cgo_import.bldpath(),
	])
	task.dynimport = cgo_obj
	task.dep_nodes = [cgo_obj]

	# 5. 6c import
	create_plan9c_task(cgo_import)

	# 6. add some of the gcc tasks to compiled_tasks
	for task in tg.compiled_tasks:
		if task.inputs[0].name.endswith('_cgo_main.c'):
			continue
		self.compiled_tasks.append(task)

	# restore the group
	self.bld.set_group(current_group)

decorate(
	feature('cgo'),
	before_method('apply_incpaths', 'propagate_uselib_vars'),
	after_method('apply_link', 'process_source'),
	process_use,
)
decorate(
	feature('cgo'),
	after_method('process_use'),
	propagate_uselib_vars,
)

def check_tool_loaded(conf, thetool):
	for tool in conf.tools:
		if tool['tool'] == thetool:
			return True
	return False

def configure(conf):
	# cgo depends on go tool
	if not check_tool_loaded(conf, 'go'):
		conf.fatal('"cgo" tool depends on "go" tool, it must be loaded first')

	# and on gcc tool
	if not check_tool_loaded(conf, 'gcc'):
		conf.fatal('"cgo" tool depends on "gcc" tool, it must be loaded first')

	# some of the go specific gcc flags
	gccflags = ['-g', '-O2']
	if conf.env.GOOS != "windows":
		gccflags.append('-fPIC')

	if conf.env.GOARCH == '386':
		gccflags.append('-m32')
	elif conf.env.GOARCH == 'amd64':
		gccflags.append('-m64')

	if conf.env.GOOS == "windows":
		gccflags.append('-mthreads')
	else:
		gccflags.append('-pthread')

	conf.env.GOGCCFLAGS = gccflags
