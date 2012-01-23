// Converting GIR type to Go/C representation

package main

import (
	"bytes"
	"fmt"
	"gobject/gi"
	"strings"
	"unsafe"
)

func ForcePointer(x string) string {
	if x == "unsafe.Pointer" {
		return x
	}
	if !strings.HasPrefix(x, "*") {
		return "*" + x
	}
	return x
}

type TypeFlags int

const (
	TypeNone    TypeFlags = 0
	TypePointer TypeFlags = 1 << iota
	TypeReturn
	TypeListMember
	TypeReceiver
	TypeExact
)

//------------------------------------------------------------------
// Cgo Type (C type in Go)
//------------------------------------------------------------------

func CgoType(ti *gi.TypeInfo, flags TypeFlags) string {
	var out bytes.Buffer

	switch tag := ti.Tag(); tag {
	case gi.TYPE_TAG_VOID:
		if ti.IsPointer() {
			out.WriteString("unsafe.Pointer")
			break
		}
		panic("Non-pointer void type is not supported in cgo")
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		out.WriteString("*C.char")
	case gi.TYPE_TAG_ARRAY:
		switch ti.ArrayType() {
		case gi.ARRAY_TYPE_C:
			out.WriteString("*")
			out.WriteString(CgoType(ti.ParamType(0), flags))
		case gi.ARRAY_TYPE_ARRAY:
			out.WriteString("*C.GArray")
		case gi.ARRAY_TYPE_PTR_ARRAY:
			out.WriteString("*C.GPtrArray")
		case gi.ARRAY_TYPE_BYTE_ARRAY:
			out.WriteString("*C.GByteArray")
		}
	case gi.TYPE_TAG_GLIST:
		out.WriteString("*C.GList")
	case gi.TYPE_TAG_GSLIST:
		out.WriteString("*C.GSList")
	case gi.TYPE_TAG_GHASH:
		out.WriteString("*C.GHashTable")
	case gi.TYPE_TAG_ERROR:
		out.WriteString("*C.GError")
	case gi.TYPE_TAG_INTERFACE:
		if ti.IsPointer() {
			flags |= TypePointer
		}
		out.WriteString(CgoTypeForInterface(ti.Interface(), flags))
	default:
		if ti.IsPointer() {
			out.WriteString("*")
		}
		out.WriteString(CgoTypeForTag(tag, flags))
	}

	return out.String()
}

func CgoTypeForInterface(bi *gi.BaseInfo, flags TypeFlags) string {
	var out bytes.Buffer

	switch bi.Type() {
	case gi.INFO_TYPE_CALLBACK:
		out.WriteString("unsafe.Pointer")
	default:
		ns := bi.Namespace()
		nm := bi.Name()
		fullnm := strings.ToLower(ns) + "." + nm

		_, disguised := GConfig.Sys.DisguisedTypes[fullnm]
		if flags&TypePointer != 0 && !disguised {
			out.WriteString("*")
		}

		out.WriteString("C.")
		out.WriteString(gi.DefaultRepository().CPrefix(ns))
		out.WriteString(bi.Name())
	}
	return out.String()
}

func CgoTypeForTag(tag gi.TypeTag, flags TypeFlags) string {
	var out bytes.Buffer
	p := PrinterTo(&out)

	if flags & TypePointer != 0 {
		p("*")
	}

	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		p("C.int")
	case gi.TYPE_TAG_INT8:
		p("C.int8_t")
	case gi.TYPE_TAG_UINT8:
		p("C.uint8_t")
	case gi.TYPE_TAG_INT16:
		p("C.int16_t")
	case gi.TYPE_TAG_UINT16:
		p("C.uint16_t")
	case gi.TYPE_TAG_INT32:
		p("C.int32_t")
	case gi.TYPE_TAG_UINT32:
		p("C.uint32_t")
	case gi.TYPE_TAG_INT64:
		p("C.int64_t")
	case gi.TYPE_TAG_UINT64:
		p("C.uint64_t")
	case gi.TYPE_TAG_FLOAT:
		p("C.float")
	case gi.TYPE_TAG_DOUBLE:
		p("C.double")
	case gi.TYPE_TAG_GTYPE:
		p("C.GType")
	case gi.TYPE_TAG_UNICHAR:
		p("C.uint32_t")
	default:
		panic("unreachable")
	}

	return out.String()
}

//------------------------------------------------------------------
// C Type
//------------------------------------------------------------------

func CTypeForTag(tag gi.TypeTag, flags TypeFlags) string {
	return CgoTypeForTag(tag, flags)[2:]
}

func CType(ti *gi.TypeInfo, flags TypeFlags) string {
	var out bytes.Buffer

	switch tag := ti.Tag(); tag {
	case gi.TYPE_TAG_VOID:
		if ti.IsPointer() {
			out.WriteString("void*")
			break
		}
		out.WriteString("void")
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		out.WriteString("char*")
	case gi.TYPE_TAG_ARRAY:
		switch ti.ArrayType() {
		case gi.ARRAY_TYPE_C:
			out.WriteString(CType(ti.ParamType(0), flags))
			out.WriteString("*")
		case gi.ARRAY_TYPE_ARRAY:
			out.WriteString("GArray*")
		case gi.ARRAY_TYPE_PTR_ARRAY:
			out.WriteString("GPtrArray*")
		case gi.ARRAY_TYPE_BYTE_ARRAY:
			out.WriteString("GByteArray*")
		}
	case gi.TYPE_TAG_GLIST:
		out.WriteString("GList*")
	case gi.TYPE_TAG_GSLIST:
		out.WriteString("GSList*")
	case gi.TYPE_TAG_GHASH:
		out.WriteString("GHashTable*")
	case gi.TYPE_TAG_ERROR:
		out.WriteString("GError*")
	case gi.TYPE_TAG_INTERFACE:
		if ti.IsPointer() {
			flags |= TypePointer
		}
		out.WriteString(CTypeForInterface(ti.Interface(), flags))
	default:
		out.WriteString(CTypeForTag(tag, flags))
		if ti.IsPointer() {
			out.WriteString("*")
		}
	}

	return out.String()
}

func CTypeForInterface(bi *gi.BaseInfo, flags TypeFlags) string {
	var out bytes.Buffer

	ns := bi.Namespace()
	nm := bi.Name()
	fullnm := strings.ToLower(ns) + "." + nm
	out.WriteString(gi.DefaultRepository().CPrefix(ns))
	out.WriteString(bi.Name())

	_, disguised := GConfig.Sys.DisguisedTypes[fullnm]
	if flags&TypePointer != 0 && !disguised {
		out.WriteString("*")
	}

	return out.String()
}

//------------------------------------------------------------------
// Go Type
//------------------------------------------------------------------

func GoTypeForInterface(bi *gi.BaseInfo, flags TypeFlags) string {
	var out bytes.Buffer
	printf := PrinterTo(&out)
	ns := bi.Namespace()
	fullnm := strings.ToLower(ns) + "." + bi.Name()

	if flags&TypeListMember != 0 {
		switch bi.Type() {
		case gi.INFO_TYPE_OBJECT, gi.INFO_TYPE_INTERFACE:
			return GoTypeForInterface(bi, TypePointer|TypeReturn)
		default:
			return GoTypeForInterface(bi, TypeReturn)
		}
	}

	switch t := bi.Type(); t {
	case gi.INFO_TYPE_OBJECT, gi.INFO_TYPE_INTERFACE:
		if flags&TypeExact != 0 {
			// exact type for object/interface is always an unsafe.Pointer
			printf("unsafe.Pointer")
			break
		}

		if flags&(TypeReturn|TypeReceiver) != 0 && flags&TypePointer != 0 {
			// receivers and return values are actual types,
			// and a pointer most likely
			printf("*")
		}
		if ns != Config.Namespace {
			// prepend foreign types with appropriate namespace
			printf("%s.", strings.ToLower(ns))
		}
		printf(bi.Name())
		if flags&(TypeReturn|TypeReceiver) == 0 {
			// ordinary function arguments are substituted by their *Like
			// counterparts
			printf("Like")
		}
		if flags&TypeReceiver != 0 && t == gi.INFO_TYPE_INTERFACE {
			// special case for interfaces, we use *Impl structures
			// as receivers
			printf("Impl")
		}
	case gi.INFO_TYPE_CALLBACK:
		if flags&TypeExact != 0 {
			printf("unsafe.Pointer")
			break
		}
		goto handle_default
	case gi.INFO_TYPE_STRUCT:
		if ns == "cairo" {
			printf(CairoGoTypeForInterface(bi, flags))
			break
		}
		goto handle_default
	default:
		goto handle_default
	}
	return out.String()
handle_default:
	_, disguised := GConfig.Sys.DisguisedTypes[fullnm]
	if flags&TypePointer != 0 && !disguised {
		printf("*")
	}
	if ns != Config.Namespace {
		printf("%s.", strings.ToLower(ns))
	}
	printf(bi.Name())
	return out.String()
}

func GoType(ti *gi.TypeInfo, flags TypeFlags) string {
	var out bytes.Buffer

	switch tag := ti.Tag(); tag {
	case gi.TYPE_TAG_VOID:
		if ti.IsPointer() {
			out.WriteString("unsafe.Pointer")
			break
		}
		panic("Non-pointer void type is not supported")
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		if flags&TypeExact != 0 {
			out.WriteString("unsafe.Pointer")
		} else {
			out.WriteString("string")
		}
	case gi.TYPE_TAG_ARRAY:
		size := ti.ArrayFixedSize()
		if size != -1 {
			fmt.Fprintf(&out, "[%d]", size)
		} else {
			if flags&TypeExact != 0 {
				out.WriteString("unsafe.Pointer")
			} else {
				out.WriteString("[]")
			}
		}
		out.WriteString(GoType(ti.ParamType(0), flags))
	case gi.TYPE_TAG_GLIST:
		if flags&TypeExact != 0 {
			out.WriteString("unsafe.Pointer")
		} else {
			out.WriteString("[]")
			out.WriteString(GoType(ti.ParamType(0), flags|TypeListMember))
		}
	case gi.TYPE_TAG_GSLIST:
		if flags&TypeExact != 0 {
			out.WriteString("unsafe.Pointer")
		} else {
			out.WriteString("[]")
			out.WriteString(GoType(ti.ParamType(0), flags|TypeListMember))
		}
	case gi.TYPE_TAG_GHASH:
		if flags&TypeExact != 0 {
			out.WriteString("unsafe.Pointer")
		} else {
			out.WriteString("map[")
			out.WriteString(GoType(ti.ParamType(0), flags))
			out.WriteString("]")
			out.WriteString(GoType(ti.ParamType(1), flags))
		}
	case gi.TYPE_TAG_ERROR:
		// not used?
		out.WriteString("error")
	case gi.TYPE_TAG_INTERFACE:
		if ti.IsPointer() {
			flags |= TypePointer
		}
		out.WriteString(GoTypeForInterface(ti.Interface(), flags))
	default:
		if ti.IsPointer() {
			flags |= TypePointer
		}
		out.WriteString(GoTypeForTag(tag, flags))
	}

	return out.String()
}

func GoTypeForTag(tag gi.TypeTag, flags TypeFlags) string {
	var out bytes.Buffer
	p := PrinterTo(&out)

	if flags & TypePointer != 0 {
		p("*")
	}

	if flags&TypeExact != 0 {
		switch tag {
		case gi.TYPE_TAG_BOOLEAN:
			p("int32") // sadly
		case gi.TYPE_TAG_INT8:
			p("int8")
		case gi.TYPE_TAG_UINT8:
			p("uint8")
		case gi.TYPE_TAG_INT16:
			p("int16")
		case gi.TYPE_TAG_UINT16:
			p("uint16")
		case gi.TYPE_TAG_INT32:
			p("int32")
		case gi.TYPE_TAG_UINT32:
			p("uint32")
		case gi.TYPE_TAG_INT64:
			p("int64")
		case gi.TYPE_TAG_UINT64:
			p("uint64")
		case gi.TYPE_TAG_FLOAT:
			p("float32")
		case gi.TYPE_TAG_DOUBLE:
			p("float64")
		case gi.TYPE_TAG_GTYPE:
			if Config.Namespace != "GObject" {
				p("gobject.Type")
			} else {
				p("Type")
			}
		case gi.TYPE_TAG_UNICHAR:
			p("rune")
		default:
			panic("unreachable")
		}
	} else {
		switch tag {
		case gi.TYPE_TAG_BOOLEAN:
			p("bool")
		case gi.TYPE_TAG_INT8:
			p("int")
		case gi.TYPE_TAG_UINT8:
			p("int")
		case gi.TYPE_TAG_INT16:
			p("int")
		case gi.TYPE_TAG_UINT16:
			p("int")
		case gi.TYPE_TAG_INT32:
			p("int")
		case gi.TYPE_TAG_UINT32:
			p("int")
		case gi.TYPE_TAG_INT64:
			p("int64")
		case gi.TYPE_TAG_UINT64:
			p("uint64")
		case gi.TYPE_TAG_FLOAT:
			p("float64")
		case gi.TYPE_TAG_DOUBLE:
			p("float64")
		case gi.TYPE_TAG_GTYPE:
			if Config.Namespace != "GObject" {
				p("gobject.Type")
			} else {
				p("Type")
			}
		case gi.TYPE_TAG_UNICHAR:
			p("rune")
		default:
			panic("unreachable")
		}
	}

	return out.String()
}

//------------------------------------------------------------------
// Simple Cgo Type (for exported functions)
//------------------------------------------------------------------

func SimpleCgoType(ti *gi.TypeInfo, flags TypeFlags) string {
	tag := ti.Tag()
	switch tag {
	case gi.TYPE_TAG_VOID:
		if ti.IsPointer() {
			return "unsafe.Pointer"
		}
		panic("Non-pointer void type is not supported")
	case gi.TYPE_TAG_INTERFACE:
		bi := ti.Interface()
		switch bi.Type() {
		case gi.INFO_TYPE_ENUM, gi.INFO_TYPE_FLAGS:
			ei := gi.ToEnumInfo(bi)
			return GoTypeForTag(ei.StorageType(), flags | TypeExact)
		case gi.INFO_TYPE_STRUCT:
			ns := bi.Namespace()
			nm := bi.Name()
			fullnm := strings.ToLower(ns) + "." + nm
			if _, ok := GConfig.Sys.DisguisedTypes[fullnm]; ok {
				return "unsafe.Pointer"
			}
		}
	}
	if !strings.HasPrefix(CgoType(ti, flags), "*") {
		return GoTypeForTag(tag, flags | TypeExact)
	}
	return "unsafe.Pointer"
}

//------------------------------------------------------------------
// Type sizes
//------------------------------------------------------------------

func TypeSizeForInterface(bi *gi.BaseInfo, flags TypeFlags) int {
	ptrsize := int(unsafe.Sizeof(unsafe.Pointer(nil)))
	if flags&TypePointer != 0 {
		return ptrsize
	}

	switch t := bi.Type(); t {
	case gi.INFO_TYPE_OBJECT, gi.INFO_TYPE_INTERFACE:
		return ptrsize
	case gi.INFO_TYPE_STRUCT:
		si := gi.ToStructInfo(bi)
		return si.Size()
	case gi.INFO_TYPE_UNION:
		ui := gi.ToUnionInfo(bi)
		return ui.Size()
	case gi.INFO_TYPE_ENUM, gi.INFO_TYPE_FLAGS:
		ei := gi.ToEnumInfo(bi)
		return TypeSizeForTag(ei.StorageType(), flags)
	case gi.INFO_TYPE_CALLBACK:
		return ptrsize
	}
	panic("unreachable: " + bi.Type().String())
}

// returns the size of a type, works only for TypeExact
func TypeSize(ti *gi.TypeInfo, flags TypeFlags) int {
	ptrsize := int(unsafe.Sizeof(unsafe.Pointer(nil)))
	switch tag := ti.Tag(); tag {
	case gi.TYPE_TAG_VOID:
		if ti.IsPointer() {
			return ptrsize
		}
		panic("Non-pointer void type is not supported")
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME, gi.TYPE_TAG_GLIST,
		gi.TYPE_TAG_GSLIST, gi.TYPE_TAG_GHASH:
		return ptrsize
	case gi.TYPE_TAG_ARRAY:
		size := ti.ArrayFixedSize()
		if size != -1 {
			return size * TypeSize(ti.ParamType(0), flags)
		}
		return ptrsize
	case gi.TYPE_TAG_INTERFACE:
		if ti.IsPointer() {
			flags |= TypePointer
		}
		return TypeSizeForInterface(ti.Interface(), flags)
	default:
		if ti.IsPointer() {
			flags |= TypePointer
		}
		return TypeSizeForTag(tag, flags)
	}
	panic("unreachable: " + ti.Tag().String())
}

func TypeSizeForTag(tag gi.TypeTag, flags TypeFlags) int {
	ptrsize := int(unsafe.Sizeof(unsafe.Pointer(nil)))
	if flags&TypePointer != 0 {
		return ptrsize
	}

	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		return 4
	case gi.TYPE_TAG_INT8:
		return 1
	case gi.TYPE_TAG_UINT8:
		return 1
	case gi.TYPE_TAG_INT16:
		return 2
	case gi.TYPE_TAG_UINT16:
		return 2
	case gi.TYPE_TAG_INT32:
		return 4
	case gi.TYPE_TAG_UINT32:
		return 4
	case gi.TYPE_TAG_INT64:
		return 8
	case gi.TYPE_TAG_UINT64:
		return 8
	case gi.TYPE_TAG_FLOAT:
		return 4
	case gi.TYPE_TAG_DOUBLE:
		return 8
	case gi.TYPE_TAG_GTYPE:
		return ptrsize
	case gi.TYPE_TAG_UNICHAR:
		return 4
	}
	panic("unreachable: " + tag.String())
}

//------------------------------------------------------------------
// Type needs wrapper?
//------------------------------------------------------------------

func TypeNeedsWrapper(ti *gi.TypeInfo) bool {
	switch tag := ti.Tag(); tag {
	case gi.TYPE_TAG_VOID:
		if ti.IsPointer() {
			return false
		}
		panic("Non-pointer void type is not supported")
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME, gi.TYPE_TAG_GLIST,
		gi.TYPE_TAG_GSLIST, gi.TYPE_TAG_GHASH:
		return true
	case gi.TYPE_TAG_ARRAY:
		size := ti.ArrayFixedSize()
		if size != -1 {
			return TypeNeedsWrapper(ti.ParamType(0))
		}
		return true
	case gi.TYPE_TAG_ERROR:
		panic("not implemented")
	case gi.TYPE_TAG_INTERFACE:
		switch ti.Interface().Type() {
		case gi.INFO_TYPE_CALLBACK, gi.INFO_TYPE_ENUM, gi.INFO_TYPE_FLAGS,
			gi.INFO_TYPE_STRUCT, gi.INFO_TYPE_UNION:
			return false
		}
		return true
	}
	return false
}
