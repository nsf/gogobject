package main

import (
	"bytes"
	"fmt"
	"gobject/gi"
	"strings"
)

func CgoArrayToGoArray(elem *gi.TypeInfo, name string) string {
	return fmt.Sprintf("(*(*[999999]%s)(unsafe.Pointer(%s)))",
		CgoType(elem, TypeNone), name)
}

type ConvFlags int

const (
	ConvNone    ConvFlags = 0
	ConvPointer ConvFlags = 1 << iota
	ConvListMember
	ConvOwnNone
	ConvOwnContainer
	ConvOwnEverything
)

func OwnershipToConvFlags(t gi.Transfer) ConvFlags {
	switch t {
	case gi.TRANSFER_NOTHING:
		return ConvOwnNone
	case gi.TRANSFER_CONTAINER:
		return ConvOwnContainer
	case gi.TRANSFER_EVERYTHING:
		return ConvOwnEverything
	}
	return 0
}

//------------------------------------------------------------------
// Go to Cgo Converter
//------------------------------------------------------------------

func GoToCgoForInterface(bi *gi.BaseInfo, arg0, arg1 string, flags ConvFlags) string {
	var out bytes.Buffer
	printf := PrinterTo(&out)

	switch bi.Type() {
	case gi.INFO_TYPE_OBJECT:
		prefix := gi.DefaultRepository().CPrefix(bi.Namespace())
		printf("if %s != nil {\n", arg0)
		printf("\t%s = %s.InheritedFrom%s%s()\n",
			arg1, arg0, prefix, bi.Name())
		printf("}")
	case gi.INFO_TYPE_ENUM, gi.INFO_TYPE_FLAGS:
		ctype := CgoTypeForInterface(bi, TypeNone)
		printf("%s = %s(%s)", arg1, ctype, arg0)
	case gi.INFO_TYPE_INTERFACE:
		prefix := gi.DefaultRepository().CPrefix(bi.Namespace())
		printf("if %s != nil {\n", arg0)
		printf("\t%s = %s.Implements%s%s()",
			arg1, arg0, prefix, bi.Name())
		printf("}")
	case gi.INFO_TYPE_STRUCT:
		ns := bi.Namespace()
		if ns == "cairo" {
			printf(CairoGoToCgoForInterface(bi, arg0, arg1, flags))
			break
		}

		fullnm := strings.ToLower(ns) + "." + bi.Name()
		if _, ok := GConfig.Sys.DisguisedTypes[fullnm]; ok {
			flags &^= ConvPointer
		}
		ctype := CgoTypeForInterface(bi, TypeNone)
		if flags&ConvPointer != 0 {
			printf("%s = (*%s)(unsafe.Pointer(%s))",
				arg1, ctype, arg0)
		} else {
			printf("%s = *(*%s)(unsafe.Pointer(&%s))",
				arg1, ctype, arg0)
		}
	case gi.INFO_TYPE_CALLBACK:
		printf("%s = unsafe.Pointer(&%s)", arg1, arg0)
	}

	return out.String()
}

func GoToCgo(ti *gi.TypeInfo, arg0, arg1 string, flags ConvFlags) string {
	var out bytes.Buffer
	printf := PrinterTo(&out)

	switch tag := ti.Tag(); tag {
	case gi.TYPE_TAG_VOID:
		if ti.IsPointer() {
			printf("%s = unsafe.Pointer(%s)", arg1, arg0)
			break
		}
		printf("<ERROR: void>")
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		printf("%s = _GoStringToGString(%s)", arg1, arg0)
		if flags&ConvOwnEverything == 0 {
			printf("\ndefer C.free(unsafe.Pointer(%s))", arg1)
		}
	case gi.TYPE_TAG_ARRAY:
		switch ti.ArrayType() {
		case gi.ARRAY_TYPE_C:
			var nelem string
			if ti.IsZeroTerminated() {
				nelem = fmt.Sprintf("(len(%s) + 1)", arg0)
			} else {
				nelem = fmt.Sprintf("len(%s)", arg0)
			}

			// alloc memory
			printf("%s = (%s)(C.malloc(C.size_t(int(unsafe.Sizeof(*%s)) * %s)))\n",
				arg1, CgoType(ti, TypeNone), arg1, nelem)
			printf("defer C.free(unsafe.Pointer(%s))\n", arg1)

			// convert elements
			printf("for i, e := range %s {\n", arg0)
			array := CgoArrayToGoArray(ti.ParamType(0), arg1)
			conv := GoToCgo(ti.ParamType(0), "e", array+"[i]", flags)
			printf(PrintLinesWithIndent(conv))
			printf("}")

			// write a trailing zero if necessary (TODO: buggy)
			if ti.IsZeroTerminated() {
				printf("\n%s[len(%s)] = nil", array, arg0)
			}

		}
	case gi.TYPE_TAG_GLIST:
	case gi.TYPE_TAG_GSLIST:
	case gi.TYPE_TAG_GHASH:
	case gi.TYPE_TAG_ERROR:
	case gi.TYPE_TAG_INTERFACE:
		if ti.IsPointer() {
			flags |= ConvPointer
		}
		printf(GoToCgoForInterface(ti.Interface(), arg0, arg1, flags))
	default:
		if ti.IsPointer() {
			flags |= ConvPointer
		}
		printf(GoToCgoForTag(tag, arg0, arg1, flags))
	}

	return out.String()
}

func GoToCgoForTag(tag gi.TypeTag, arg0, arg1 string, flags ConvFlags) string {
	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		return fmt.Sprintf("%s = _GoBoolToCBool(%s)", arg1, arg0)
	default:
		if flags & ConvPointer == 0 {
			return fmt.Sprintf("%s = %s(%s)", arg1,
				CgoTypeForTag(tag, TypeNone), arg0)
		} else {
			return fmt.Sprintf("%s = (%s)(unsafe.Pointer(%s))", arg1,
				CgoTypeForTag(tag, TypePointer), arg0)
		}
	}

	panic("unreachable")
	return ""
}

//------------------------------------------------------------------
// Cgo to Go Converter
//------------------------------------------------------------------

func CgoToGoForInterface(bi *gi.BaseInfo, arg1, arg2 string, flags ConvFlags) string {
	var out bytes.Buffer
	printf := PrinterTo(&out)

	switch bi.Type() {
	case gi.INFO_TYPE_OBJECT, gi.INFO_TYPE_INTERFACE:
		gotype := GoTypeForInterface(bi, TypeReturn)
		if flags&ConvOwnEverything != 0 {
			printf("%s = (*%s)(_GObjectWrap(unsafe.Pointer(%s)))",
				arg2, gotype, arg1)
		} else {
			printf("%s = (*%s)(_GObjectGrab(unsafe.Pointer(%s)))",
				arg2, gotype, arg1)
		}
	case gi.INFO_TYPE_ENUM, gi.INFO_TYPE_FLAGS:
		gotype := GoTypeForInterface(bi, TypeReturn)
		printf("%s = %s(%s)", arg2, gotype, arg1)
	case gi.INFO_TYPE_STRUCT, gi.INFO_TYPE_UNION:
		ns := bi.Namespace()
		if ns == "cairo" {
			printf(CairoCgoToGoForInterface(bi, arg1, arg2, flags))
			break
		}

		fullnm := strings.ToLower(ns) + "." + bi.Name()
		gotype := GoTypeForInterface(bi, TypeReturn)

		if flags&ConvListMember != 0 {
			printf("%s = *(*%s)(unsafe.Pointer(%s))",
				arg2, gotype, arg1)
			break
		}

		if _, ok := GConfig.Sys.DisguisedTypes[fullnm]; ok {
			printf("%s = %s{unsafe.Pointer(%s)}",
				arg2, gotype, arg1)
			break
		}

		if flags&ConvPointer != 0 {
			printf("%s = (*%s)(unsafe.Pointer(%s))",
				arg2, gotype, arg1)
		} else {
			printf("%s = *(*%s)(unsafe.Pointer(&%s))",
				arg2, gotype, arg1)
		}
	}
	return out.String()
}

func CgoToGo(ti *gi.TypeInfo, arg1, arg2 string, flags ConvFlags) string {
	var out bytes.Buffer
	printf := PrinterTo(&out)

	switch tag := ti.Tag(); tag {
	case gi.TYPE_TAG_VOID:
		if ti.IsPointer() {
			printf("%s = %s", arg2, arg1)
			break
		}
		printf("<ERROR: void>")
	case gi.TYPE_TAG_UTF8, gi.TYPE_TAG_FILENAME:
		printf("%s = C.GoString(%s)", arg2, arg1)
		if flags&ConvOwnEverything != 0 {
			printf("\nC.g_free(unsafe.Pointer(%s))", arg1)
		}
	case gi.TYPE_TAG_ARRAY:
		switch ti.ArrayType() {
		case gi.ARRAY_TYPE_C:
			// array was allocated already at this point
			printf("for i := range %s {\n", arg2)
			array := CgoArrayToGoArray(ti.ParamType(0), arg1)
			conv := CgoToGo(ti.ParamType(0),
				array+"[i]", arg2+"[i]", flags)
			printf(PrintLinesWithIndent(conv))
			printf("}")
		}
	case gi.TYPE_TAG_GLIST:
		ptype := ti.ParamType(0)
		printf("for iter := (*_GList)(unsafe.Pointer(%s)); iter != nil; iter = iter.next {\n", arg1)
		elt := fmt.Sprintf("(%s)(iter.data)",
			ForcePointer(CgoType(ptype, TypeReturn|TypeListMember)))
		printf("\tvar elt %s\n", GoType(ptype, TypeReturn|TypeListMember))
		conv := CgoToGo(ptype, elt, "elt", flags|ConvListMember)
		printf(PrintLinesWithIndent(conv))
		printf("\t%s = append(%s, elt)\n", arg2, arg2)
		printf("}")
	case gi.TYPE_TAG_GSLIST:
		ptype := ti.ParamType(0)
		printf("for iter := (*_GSList)(unsafe.Pointer(%s)); iter != nil; iter = iter.next {\n", arg1)
		elt := fmt.Sprintf("(%s)(iter.data)",
			ForcePointer(CgoType(ptype, TypeReturn|TypeListMember)))
		printf("\tvar elt %s\n", GoType(ptype, TypeReturn|TypeListMember))
		conv := CgoToGo(ptype, elt, "elt", flags|ConvListMember)
		printf(PrintLinesWithIndent(conv))
		printf("\t%s = append(%s, elt)\n", arg2, arg2)
		printf("}")
	case gi.TYPE_TAG_GHASH:
	case gi.TYPE_TAG_ERROR:
		printf("if %s != nil {\n", arg1)
		printf("\t%s = errors.New(C.GoString(((*_GError)(unsafe.Pointer(%s))).message))\n", arg2, arg1)
		printf("\tC.g_error_free(%s)\n", arg1)
		printf("}\n")
	case gi.TYPE_TAG_INTERFACE:
		if ti.IsPointer() {
			flags |= ConvPointer
		}
		printf(CgoToGoForInterface(ti.Interface(), arg1, arg2, flags))
	default:
		if ti.IsPointer() {
			flags |= ConvPointer
		}
		printf(CgoToGoForTag(tag, arg1, arg2, flags))
	}

	return out.String()
}

func CgoToGoForTag(tag gi.TypeTag, arg1, arg2 string, flags ConvFlags) string {
	switch tag {
	case gi.TYPE_TAG_BOOLEAN:
		return fmt.Sprintf("%s = %s != 0", arg2, arg1)
	case gi.TYPE_TAG_INT8:
		return fmt.Sprintf("%s = int(%s)", arg2, arg1)
	case gi.TYPE_TAG_UINT8:
		return fmt.Sprintf("%s = int(%s)", arg2, arg1)
	case gi.TYPE_TAG_INT16:
		return fmt.Sprintf("%s = int(%s)", arg2, arg1)
	case gi.TYPE_TAG_UINT16:
		return fmt.Sprintf("%s = int(%s)", arg2, arg1)
	case gi.TYPE_TAG_INT32:
		return fmt.Sprintf("%s = int(%s)", arg2, arg1)
	case gi.TYPE_TAG_UINT32:
		return fmt.Sprintf("%s = int(%s)", arg2, arg1)
	case gi.TYPE_TAG_INT64:
		return fmt.Sprintf("%s = int64(%s)", arg2, arg1)
	case gi.TYPE_TAG_UINT64:
		return fmt.Sprintf("%s = uint64(%s)", arg2, arg1)
	case gi.TYPE_TAG_FLOAT:
		return fmt.Sprintf("%s = float64(%s)", arg2, arg1)
	case gi.TYPE_TAG_DOUBLE:
		return fmt.Sprintf("%s = float64(%s)", arg2, arg1)
	case gi.TYPE_TAG_GTYPE:
		if Config.Namespace != "GObject" {
			return fmt.Sprintf("%s = gobject.Type(%s)", arg2, arg1)
		}
		return fmt.Sprintf("%s = Type(%s)", arg2, arg1)
	case gi.TYPE_TAG_UNICHAR:
		return fmt.Sprintf("%s = rune(%s)", arg2, arg1)
	}

	panic("unreachable")
	return ""
}

//------------------------------------------------------------------
// Simple Cgo to Go Converter
//------------------------------------------------------------------

func SimpleCgoToGo(ti *gi.TypeInfo, arg0, arg1 string, flags ConvFlags) string {
	cgotype := CgoType(ti, TypeNone)
	arg0 = fmt.Sprintf("(%s)(%s)", cgotype, arg0)
	return CgoToGo(ti, arg0, arg1, flags)
}
