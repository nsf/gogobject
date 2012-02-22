from waflib.Task import Task
from waflib.TaskGen import extension

class gogobject(Task):
	run_str = '${GGG} ${GGGFLAGS} -o ${TGT} ${SRC}'

@extension('.go.in')
def gogobject_hook(self, node):
	if not self.env.GGG:
		tg = self.bld.get_tgen_by_name('go-gobject-gen')
		ggg = tg.link_task.outputs[0]
		self.env.GGG = ggg.abspath()

	task = self.create_task('gogobject', node, node.change_ext(''))
	return task
