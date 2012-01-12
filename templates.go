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
	if x == "\x00" {
		return nil
	}
	return C.CString(x)
}

func _GoBoolToCBool(x bool) C.int {
	if x { return 1 }
	return 0
}

func _CInterfaceToGoInterface(iface [2]unsafe.Pointer) interface{} {
	return *(*interface{})(unsafe.Pointer(&iface))
}

func _GoInterfaceToCInterface(iface interface{}) *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Pointer(&iface))
}

var _cbcache = make(map[unsafe.Pointer]bool)

//export _[<.namespace>]_go_callback_cleanup
func _[<.namespace>]_go_callback_cleanup(gofunc unsafe.Pointer) {
	delete(_cbcache, gofunc)
}
`)

var ObjectTemplate = MustTemplate(`
type [<.name>]Like interface {
	[<.parentlike>]
	InheritedFrom[<.cprefix>][<.name>]() [<.cgotype>]
}

type [<.name>] struct {
	[<.parent>]
	[<.interfaces>]
}

func To[<.name>](objlike [<.gobjectns>]ObjectLike) *[<.name>] {
	c := objlike.InheritedFromGObject()
	if c == nil {
		return nil
	}
	t := (*[<.name>])(nil).GetStaticType()
	obj := [<.gobjectns>]ObjectGrabIfType(unsafe.Pointer(c), t)
	if obj != nil {
		return (*[<.name>])(obj)
	}
	panic("cannot cast to [<.name>]")
}

func (this0 *[<.name>]) InheritedFrom[<.cprefix>][<.name>]() [<.cgotype>] {
	if this0 == nil {
		return nil
	}
	return ([<.cgotype>])(this0.C)
}

func (this0 *[<.name>]) GetStaticType() [<.gobjectns>]Type {
	return [<.gobjectns>]Type(C.[<.typeinit>]())
}
`)

// XXX: uses gc specific hack, expect problems on gccgo and/or ask developers
// about the address of an empty embedded struct
var InterfaceTemplate = MustTemplate(`
type [<.name>]Like interface {
	Implements[<.cprefix>][<.name>]() [<.cgotype>]
}

type [<.name>] struct {
	[<.gobjectns>]Object
	[<.name>]Impl
}

type [<.name>]Impl struct {}

func To[<.name>](objlike [<.gobjectns>]ObjectLike) *[<.name>] {
	t := (*[<.name>]Impl)(nil).GetStaticType()
	c := objlike.InheritedFromGObject()
	obj := [<.gobjectns>]ObjectGrabIfType(unsafe.Pointer(c), t)
	if obj != nil {
		return (*[<.name>])(obj)
	}
	panic("cannot cast to [<.name>]")
}

func (this0 *[<.name>]Impl) Implements[<.cprefix>][<.name>]() [<.cgotype>] {
	obj := uintptr(unsafe.Pointer(this0)) - unsafe.Sizeof(uintptr(0))
	return ([<.cgotype>])((*[<.gobjectns>]Object)(unsafe.Pointer(obj)).C)
}

func (this0 *[<.name>]Impl) GetStaticType() [<.gobjectns>]Type {
	return [<.gobjectns>]Type(C.[<.typeinit>]())
}

`)

var CUtilsTemplate = MustTemplate(`
extern void _[<.namespace>]_go_callback_cleanup(void *gofunc);
static void _c_callback_cleanup(void *userdata)
{
	_[<.namespace>]_go_callback_cleanup(userdata);
}
`)
