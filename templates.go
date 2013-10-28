package main

const g_object_ref_unref = `extern GObject *g_object_ref_sink(GObject*);
extern void g_object_unref(GObject*);`

const g_error_free = `extern void g_error_free(GError*);`

const g_free = `extern void g_free(void*);`

const _array_length = `unsigned int _array_length(void* _array) { void** array = _array; unsigned int i = 0; while(array && array[i] != 0) i++; return i;}`

var go_utils_template = must_template(`
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

[<if not .nocallbacks>]
//export _[<.namespace>]_go_callback_cleanup
func _[<.namespace>]_go_callback_cleanup(gofunc unsafe.Pointer) {
	[<.gobjectns>]Holder.Release(gofunc)
}
[<end>]
`)

var object_template = must_template(`
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

func [<.name>]GetType() [<.gobjectns>]Type {
	return (*[<.name>])(nil).GetStaticType()
}
`)

// XXX: uses gc specific hack, expect problems on gccgo and/or ask developers
// about the address of an empty embedded struct
var interface_template = must_template(`
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

func [<.name>]GetType() [<.gobjectns>]Type {
	return (*[<.name>]Impl)(nil).GetStaticType()
}
`)

const c_header = `#pragma once
#include <stdlib.h>
#include <stdint.h>

typedef size_t GType;
typedef void *GVaClosureMarshal;

`

var c_template = must_template(`
#include "[<.package>].gen.h"

static void _c_callback_cleanup(void *userdata)
{
	_[<.namespace>]_go_callback_cleanup(userdata);
}
`)
