// Converting GIR type to Go/C representation

package main

import (
	"bytes"
	"fmt"
	"gobject/gi"
	"strings"
)

func ForcePointer(x string) string {
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

	ns := bi.Namespace()
	nm := bi.Name()
	fullnm := strings.ToLower(ns) + "." + nm

	_, disguised := GConfig.Sys.DisguisedTypes[fullnm]
	if flags&TypePointer != 0 && !disguised {
		out.WriteString("*")
	}

	out.WriteString("C.")
	if ctype, ok := GConfig.CTypes[ns+"."+nm]; ok {
		out.WriteString(ctype)
	} else {
		out.WriteString(gi.DefaultRepository().CPrefix(ns))
		out.WriteString(bi.Name())
	}

	return out.String()
}

func CgoTypeForTag(tag gi.TypeTag, flags TypeFlags) string {
	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		return "C.int"
	case gi.TYPE_TAG_INT8:
		return "C.int8_t"
	case gi.TYPE_TAG_UINT8:
		return "C.uint8_t"
	case gi.TYPE_TAG_INT16:
		return "C.int16_t"
	case gi.TYPE_TAG_UINT16:
		return "C.uint16_t"
	case gi.TYPE_TAG_INT32:
		return "C.int32_t"
	case gi.TYPE_TAG_UINT32:
		return "C.uint32_t"
	case gi.TYPE_TAG_INT64:
		return "C.int64_t"
	case gi.TYPE_TAG_UINT64:
		return "C.uint64_t"
	case gi.TYPE_TAG_FLOAT:
		return "C.float"
	case gi.TYPE_TAG_DOUBLE:
		return "C.double"
	case gi.TYPE_TAG_GTYPE:
		return "C.GType"
	case gi.TYPE_TAG_UNICHAR:
		return "C.uint32_t"
	}

	panic("unreachable")
	return ""
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
	if ctype, ok := GConfig.CTypes[fullnm]; ok {
		out.WriteString(ctype)
	} else {
		out.WriteString(gi.DefaultRepository().CPrefix(ns))
		out.WriteString(bi.Name())
	}
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
		case gi.INFO_TYPE_OBJECT:
			return GoTypeForInterface(bi, TypePointer|TypeReturn)
		default:
			return GoTypeForInterface(bi, TypeReturn)
		}
	}

	switch {
	case bi.Type() == gi.INFO_TYPE_OBJECT && flags&TypeReturn == 0:
		oi := gi.ToObjectInfo(bi)
		if ns != Config.Namespace {
			printf("%s.", strings.ToLower(ns))
		}
		printf("%sLike", oi.Name())
	case bi.Type() == gi.INFO_TYPE_INTERFACE:
		if ns != Config.Namespace {
			printf("%s.", strings.ToLower(ns))
		}
		out.WriteString(bi.Name())
	default:
		_, disguised := GConfig.Sys.DisguisedTypes[fullnm]
		if flags&TypePointer != 0 && !disguised {
			out.WriteString("*")
		}
		if ns != Config.Namespace {
			printf("%s.", strings.ToLower(ns))
		}
		out.WriteString(bi.Name())
	}
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
		out.WriteString("string")
	case gi.TYPE_TAG_ARRAY:
		size := ti.ArrayFixedSize()
		if size != -1 {
			fmt.Fprintf(&out, "[%d]", size)
		} else {
			out.WriteString("[]")
		}
		out.WriteString(GoType(ti.ParamType(0), flags))
	case gi.TYPE_TAG_GLIST:
		out.WriteString("[]")
		out.WriteString(GoType(ti.ParamType(0), flags|TypeListMember))
	case gi.TYPE_TAG_GSLIST:
		out.WriteString("[]")
		out.WriteString(GoType(ti.ParamType(0), flags|TypeListMember))
	case gi.TYPE_TAG_GHASH:
		out.WriteString("map[")
		out.WriteString(GoType(ti.ParamType(0), flags))
		out.WriteString("]")
		out.WriteString(GoType(ti.ParamType(1), flags))
	case gi.TYPE_TAG_ERROR:
		out.WriteString("error")
	case gi.TYPE_TAG_INTERFACE:
		if ti.IsPointer() {
			flags |= TypePointer
		}
		out.WriteString(GoTypeForInterface(ti.Interface(), flags))
	default:
		if ti.IsPointer() {
			out.WriteString("*")
		}
		out.WriteString(GoTypeForTag(tag, flags))
	}

	return out.String()
}

func GoTypeForTag(tag gi.TypeTag, flags TypeFlags) string {
	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		return "bool"
	case gi.TYPE_TAG_INT8:
		return "int"
	case gi.TYPE_TAG_UINT8:
		return "int"
	case gi.TYPE_TAG_INT16:
		return "int"
	case gi.TYPE_TAG_UINT16:
		return "int"
	case gi.TYPE_TAG_INT32:
		return "int"
	case gi.TYPE_TAG_UINT32:
		return "int"
	case gi.TYPE_TAG_INT64:
		return "int64"
	case gi.TYPE_TAG_UINT64:
		return "uint64"
	case gi.TYPE_TAG_FLOAT:
		return "float64"
	case gi.TYPE_TAG_DOUBLE:
		return "float64"
	case gi.TYPE_TAG_GTYPE:
		if Config.Namespace != "GObject" {
			return "gobject.Type"
		}
		return "Type"
	case gi.TYPE_TAG_UNICHAR:
		return "rune"
	}

	panic("unreachable")
	return ""
}
