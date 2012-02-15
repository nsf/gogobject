package main

import (
	"github.com/nsf/gogobject/gi"
	"bytes"
)

func CairoGoTypeForInterface(bi *gi.BaseInfo, flags TypeFlags) string {
	var out bytes.Buffer
	p := PrinterTo(&out)
	name := bi.Name()

	switch name {
	case "Surface", "Pattern":
		if flags&(TypeReturn) == 0 {
			p("cairo.%sLike", name)
			break
		}
		fallthrough
	default:
		if flags&TypePointer != 0 {
			p("*")
		}
		p("cairo.%s", name)
	}

	return out.String()
}

func CairoGoToCgoForInterface(bi *gi.BaseInfo, arg0, arg1 string, flags ConvFlags) string {
	var out bytes.Buffer
	p := PrinterTo(&out)
	name := bi.Name()

	switch name {
	case "Surface", "Pattern":
		p("if %s != nil {\n", arg0)
		p("\t%s = (*C.cairo%s)(%s.InheritedFromCairo%s().C)\n", arg1, name, arg0, name)
		p("}")
	case "RectangleInt", "Rectangle", "TextCluster", "Matrix":
		ctype := CgoTypeForInterface(bi, TypeNone)
		if flags&ConvPointer != 0 {
			p("%s = (*%s)(unsafe.Pointer(%s))",
				arg1, ctype, arg0)
		} else {
			p("%s = *(*%s)(unsafe.Pointer(&%s))",
				arg1, ctype, arg0)
		}
	default:
		p("if %s != nil {\n", arg0)
		p("\t%s = (*C.cairo%s)(%s.C)\n", arg1, name, arg0)
		p("}")
	}

	return out.String()
}

func CairoCgoToGoForInterface(bi *gi.BaseInfo, arg1, arg2 string, flags ConvFlags) string {
	var out bytes.Buffer
	p := PrinterTo(&out)
	name := bi.Name()

	switch name {
	case "Path", "FontOptions":
		if flags&ConvPointer == 0 {
			panic("unexpected non-pointer type")
		}

		p("%s = (*cairo.%s)(cairo.%sWrap(unsafe.Pointer(%s)))", arg2, name, name, arg1)
	case "Surface", "Region", "Pattern", "Context", "FontFace", "ScaledFont":
		if flags&ConvPointer == 0 {
			panic("unexpected non-pointer type")
		}

		grab := "true"
		if flags&ConvOwnEverything != 0 {
			grab = "false"
		}
		p("%s = (*cairo.%s)(cairo.%sWrap(unsafe.Pointer(%s), %s))", arg2, name, name, arg1, grab)
	default:
		gotype := GoTypeForInterface(bi, TypeReturn)
		if flags&ConvPointer != 0 {
			p("%s = (*%s)(unsafe.Pointer(%s))",
				arg2, gotype, arg1)
		} else {
			p("%s = *(*%s)(unsafe.Pointer(&%s))",
				arg2, gotype, arg1)
		}
	}

	return out.String()
}