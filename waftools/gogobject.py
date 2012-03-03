from waflib.Task import Task
from waflib.TaskGen import extension

class gogobject(Task):
	run_str = '${GGG} ${GGGFLAGS} -o ${TGT[0].parent.abspath()} ${SRC}'

@extension('.go.in')
def gogobject_hook(self, node):
	tg = self.bld.get_tgen_by_name('go-gobject-gen')
	ggg = tg.link_task.outputs[0]

	if not self.env.GGG:
		self.env.GGG = ggg.abspath()

	go_out = node.change_ext('')
	c_out = go_out.change_ext('.gen.c')
	h_out = go_out.change_ext('.gen.h')
	task = self.create_task('gogobject', node, [go_out, c_out, h_out])
	task.dep_nodes = [ggg]
	return task
