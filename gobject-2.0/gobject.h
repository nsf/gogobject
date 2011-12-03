#pragma once

typedef struct _GGoClosure GGoClosure;

GType _g_object_type(GObject *obj);
GType _g_value_type(GValue *val);
GType _g_type_interface();
GType _g_type_char();
GType _g_type_uchar();
GType _g_type_boolean();
GType _g_type_int();
GType _g_type_uint();
GType _g_type_long();
GType _g_type_ulong();
GType _g_type_int64();
GType _g_type_uint64();
GType _g_type_enum();
GType _g_type_flags();
GType _g_type_float();
GType _g_type_double();
GType _g_type_string();
GType _g_type_pointer();
GType _g_type_boxed();
GType _g_type_param();
GType _g_type_object();
GType _g_type_gtype();
GType _g_type_variant();

GGoClosure *g_goclosure_new(int32_t id, void **iface);
int32_t g_goclosure_get_id(GGoClosure *clo);
void g_goclosure_get_iface(GGoClosure *clo, void **out);
