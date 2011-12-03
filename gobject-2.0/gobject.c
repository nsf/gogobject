#include <glib-object.h>
#include <stdint.h>
#include "gobject.h"

struct _GGoClosure {
	GClosure closure;
	int32_t id;
	void *iface[2];
};

GType _g_object_type(GObject *obj) {
	return G_OBJECT_TYPE(obj);
}

GType _g_value_type(GValue *val) {
	return G_VALUE_TYPE(val);
}

GType _g_type_interface()	{ return G_TYPE_INTERFACE; }
GType _g_type_char()		{ return G_TYPE_CHAR; }
GType _g_type_uchar()		{ return G_TYPE_UCHAR; }
GType _g_type_boolean()		{ return G_TYPE_BOOLEAN; }
GType _g_type_int()		{ return G_TYPE_INT; }
GType _g_type_uint()		{ return G_TYPE_UINT; }
GType _g_type_long()		{ return G_TYPE_LONG; }
GType _g_type_ulong()		{ return G_TYPE_ULONG; }
GType _g_type_int64()		{ return G_TYPE_INT64; }
GType _g_type_uint64()		{ return G_TYPE_UINT64; }
GType _g_type_enum()		{ return G_TYPE_ENUM; }
GType _g_type_flags()		{ return G_TYPE_FLAGS; }
GType _g_type_float()		{ return G_TYPE_FLOAT; }
GType _g_type_double()		{ return G_TYPE_DOUBLE; }
GType _g_type_string()		{ return G_TYPE_STRING; }
GType _g_type_pointer()		{ return G_TYPE_POINTER; }
GType _g_type_boxed()		{ return G_TYPE_BOXED; }
GType _g_type_param()		{ return G_TYPE_PARAM; }
GType _g_type_object()		{ return G_TYPE_OBJECT; }
GType _g_type_gtype()		{ return G_TYPE_GTYPE; }
GType _g_type_variant()		{ return G_TYPE_VARIANT; }

extern void g_goclosure_marshal_go(GGoClosure*, GValue*, int32_t, GValue*);
extern void g_goclosure_finalize_go(GGoClosure*);

static void g_goclosure_finalize(void *notify_data, GClosure *closure)
{
	GGoClosure *goclosure = (GGoClosure*)closure;
	g_goclosure_finalize_go(goclosure);
}

static void g_goclosure_marshal(GClosure *closure, GValue *return_value,
				uint32_t n_param_values, const GValue *param_values,
				void *invocation_hint, void *data)
{
	g_goclosure_marshal_go((GGoClosure*)closure,
			       return_value,
			       n_param_values,
			       (GValue*)param_values);
}

GGoClosure *g_goclosure_new(int32_t id, void **iface)
{
	GClosure *closure;
	GGoClosure *goclosure;

	closure = g_closure_new_simple(sizeof(GGoClosure), 0);
	goclosure = (GGoClosure*)closure;
	goclosure->id = id;
	goclosure->iface[0] = iface[0];
	goclosure->iface[1] = iface[1];

	g_closure_add_finalize_notifier(closure, 0, g_goclosure_finalize);
	g_closure_set_marshal(closure, g_goclosure_marshal);

	return goclosure;
}

int32_t g_goclosure_get_id(GGoClosure *clo) {
	return clo->id;
}

void g_goclosure_get_iface(GGoClosure *clo, void **out) {
	out[0] = clo->iface[0];
	out[1] = clo->iface[1];
}


