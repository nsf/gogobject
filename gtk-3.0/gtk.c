#include <gtk/gtk.h>
#include <stdio.h>
#include "gtk.h"

GParamSpec *_gtk_container_find_child_property(GtkContainer *container, const char *name)
{
	GObjectClass *cls = G_OBJECT_GET_CLASS(container);
	return gtk_container_class_find_child_property(cls, name);
}
