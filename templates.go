package main

const CommonIncludes = `#include <stdlib.h>
#include <stdint.h>`

const GType = `typedef size_t GType;`

const GObjectRefUnref = `extern GObject *g_object_ref_sink(GObject*);
extern void g_object_unref(GObject*);`

const GErrorFree = `extern void g_error_free(GError*);`

const GFree = `extern void g_free(void*);`

var GoUtilsTemplate = MustTemplate(`
const alot = 999999

type _GSList struct {
	data unsafe.Pointer
	next *_GSList
}

type _GList struct {
	data unsafe.Pointer
	next *_GList
	prev *_GList
}

type _GError struct {
	domain uint32
	code int32
	message *C.char
}

func _GoStringToGString(x string) *C.char {
	if x == "" {
		return nil
	}
	return C.CString(x)
}

func _GoBoolToCBool(x bool) C.int {
	if x { return 1 }
	return 0
}

func _GObjectFinalizer(obj *[<.gobject>]) {
	C.g_object_unref((*C.GObject)(obj.C))
}

func _SetGObjectFinalizer(obj *[<.gobject>]) {
	runtime.SetFinalizer(obj, _GObjectFinalizer)
}

// returns a new *gobject.Object casted to unsafe.Pointer, so that various
// callers can cast it back to any inherited type, note that function sets
// finalizer as well
func _GObjectGrab(c unsafe.Pointer) unsafe.Pointer {
	if c == nil {
		return nil
	}
	obj := &[<.gobject>]{c}
	C.g_object_ref_sink((*C.GObject)(obj.C))
	_SetGObjectFinalizer(obj)
	return unsafe.Pointer(obj)
}

// same as above, but doesn't increment reference count
func _GObjectWrap(c unsafe.Pointer) unsafe.Pointer {
	if c == nil {
		return nil
	}
	obj := &[<.gobject>]{c}
	_SetGObjectFinalizer(obj)
	return unsafe.Pointer(obj)
}

func _CInterfaceToGoInterface(iface [2]unsafe.Pointer) interface{} {
	return *(*interface{})(unsafe.Pointer(&iface))
}

func _GoInterfaceToCInterface(iface interface{}) *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Pointer(&iface))
}

func _GObjectGrabIfType(c unsafe.Pointer, t [<.gtype>]) unsafe.Pointer {
	if c == nil {
		return nil
	}
	obj := &[<.gobject>]{c}
	if obj.GetType().IsA(t) {
		C.g_object_ref_sink((*C.GObject)(obj.C))
		_SetGObjectFinalizer(obj)
		return unsafe.Pointer(obj)
	}
	return nil
}

var _cbcache = make(map[unsafe.Pointer]bool)

//export _[<.namespace>]_go_callback_cleanup
func _[<.namespace>]_go_callback_cleanup(gofunc unsafe.Pointer) {
	delete(_cbcache, gofunc)
}
`)

var ObjectTemplate = MustTemplate(`
type [<.name>]Like interface {
	InheritedFrom[<.cprefix>][<.name>]() [<.cgotype>]
}

type [<.name>] struct {
	[<.parent>]
	[<.interfaces>]
}

func To[<.name>](objlike [<.gobject>]Like) *[<.name>] {
	t := (*[<.name>])(nil).GetStaticType()
	c := objlike.InheritedFromGObject()
	obj := _GObjectGrabIfType(unsafe.Pointer(c), t)
	if obj != nil {
		return (*[<.name>])(obj)
	}
	panic("cannot cast to [<.name>]")
}

func (this0 *[<.name>]) InheritedFrom[<.cprefix>][<.name>]() [<.cgotype>] {
	return ([<.cgotype>])(this0.C)
}

func (this0 *[<.name>]) GetStaticType() [<.gtype>] {
	return [<.gtype>](C.[<.typeinit>]())
}
`)

// XXX: uses gc specific hack, expect problems on gccgo and/or ask developers
// about the address of an empty embedded struct
var InterfaceTemplate = MustTemplate(`
type [<.name>]Like interface {
	Implements[<.cprefix>][<.name>]() [<.cgotype>]
}

type [<.name>] struct {
	[<.gobject>]
	[<.name>]Impl
}

type [<.name>]Impl struct {}

func To[<.name>](objlike [<.gobject>]Like) *[<.name>] {
	t := (*[<.name>]Impl)(nil).GetStaticType()
	c := objlike.InheritedFromGObject()
	obj := _GObjectGrabIfType(unsafe.Pointer(c), t)
	if obj != nil {
		return (*[<.name>])(obj)
	}
	panic("cannot cast to [<.name>]")
}

func (this0 *[<.name>]Impl) Implements[<.cprefix>][<.name>]() [<.cgotype>] {
	obj := uintptr(unsafe.Pointer(this0)) - unsafe.Sizeof(uintptr(0))
	return ([<.cgotype>])((*[<.gobject>])(unsafe.Pointer(obj)).C)
}

func (this0 *[<.name>]Impl) GetStaticType() [<.gtype>] {
	return [<.gtype>](C.[<.typeinit>]())
}

`)

var CUtilsTemplate = MustTemplate(`
extern void _[<.namespace>]_go_callback_cleanup(void *gofunc);
static void _c_callback_cleanup(void *userdata)
{
	_[<.namespace>]_go_callback_cleanup(userdata);
}
`)
