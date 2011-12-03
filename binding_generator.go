package main

import (
	"bytes"
	"fmt"
	"gobject/gi"
	"io"
	"strings"
	"unsafe"
)

var printf func(string, ...interface{})

func PrinterTo(w io.Writer) func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		fmt.Fprintf(w, format, args...)
	}
}

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
}`)

func CFuncForwardDeclaration(fi *gi.FunctionInfo, container *gi.BaseInfo) {
	flags := fi.Flags()
	narg := fi.NumArg()
	if flags&gi.FUNCTION_THROWS != 0 {
		narg++
	}

	printf("extern %s %s(", CType(fi.ReturnType(), TypeNone), fi.Symbol())
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("%s", CTypeForInterface(container, TypePointer))
		if narg > 0 {
			printf(", ")
		}
	}

	for i, n := 0, narg; i < n; i++ {
		if i != 0 {
			printf(", ")
		}

		if i == n-1 && flags&gi.FUNCTION_THROWS != 0 {
			printf("GError**")
			continue
		}

		arg := fi.Arg(i)
		printf("%s", CType(arg.Type(), TypeNone))

		switch arg.Direction() {
		case gi.DIRECTION_INOUT, gi.DIRECTION_OUT:
			printf("*")
		}
	}
	printf(");\n")
}

// generating forward C declarations properly:
// 10 - various typedefs
// 20 - functions and methods
// 30 - struct definitions

func CForwardDeclaration10(bi *gi.BaseInfo) {
	switch bi.Type() {
	case gi.INFO_TYPE_OBJECT:
		cctype := CTypeForInterface(bi, TypeNone)
		printf("typedef struct _%s %s;\n", cctype, cctype)
	case gi.INFO_TYPE_FLAGS, gi.INFO_TYPE_ENUM:
		ei := gi.ToEnumInfo(bi)
		printf("typedef %s %s;\n", CTypeForTag(ei.StorageType(), TypeNone),
			CTypeForInterface(bi, TypeNone))
	case gi.INFO_TYPE_STRUCT, gi.INFO_TYPE_INTERFACE, gi.INFO_TYPE_UNION:
		cctype := CTypeForInterface(bi, TypeNone)
		printf("typedef struct _%s %s;\n", cctype, cctype)
	case gi.INFO_TYPE_CALLBACK:
		// TODO
		printf("typedef void* %s;\n", CTypeForInterface(bi, TypeNone))
	}
}

func CForwardDeclaration20(bi *gi.BaseInfo) {
	switch bi.Type() {
	case gi.INFO_TYPE_FUNCTION:
		fi := gi.ToFunctionInfo(bi)
		CFuncForwardDeclaration(fi, nil)
	case gi.INFO_TYPE_OBJECT:
		oi := gi.ToObjectInfo(bi)
		for i, n := 0, oi.NumMethod(); i < n; i++ {
			meth := oi.Method(i)
			CFuncForwardDeclaration(meth, bi)
		}
		printf("extern GType %s();\n", oi.TypeInit())
	case gi.INFO_TYPE_STRUCT:
		si := gi.ToStructInfo(bi)
		for i, n := 0, si.NumMethod(); i < n; i++ {
			meth := si.Method(i)
			CFuncForwardDeclaration(meth, bi)
		}
	case gi.INFO_TYPE_UNION:
		ui := gi.ToUnionInfo(bi)
		for i, n := 0, ui.NumMethod(); i < n; i++ {
			meth := ui.Method(i)
			CFuncForwardDeclaration(meth, bi)
		}
	}
}

func CForwardDeclaration30(bi *gi.BaseInfo) {
	fullnm := strings.ToLower(bi.Namespace()) + "." + bi.Name()

	switch bi.Type() {
	case gi.INFO_TYPE_STRUCT:
		si := gi.ToStructInfo(bi)
		size := si.Size()
		if _, ok := GConfig.Sys.DisguisedTypes[fullnm]; ok {
			size = int(unsafe.Sizeof(unsafe.Pointer(nil)))
		}
		cctype := CTypeForInterface(bi, TypeNone)
		if size == 0 {
			printf("struct _%s {};\n", cctype)
		} else {
			printf("struct _%s { uint8_t _data[%d]; };\n", cctype, size)
		}
	case gi.INFO_TYPE_UNION:
		ui := gi.ToUnionInfo(bi)
		size := ui.Size()
		cctype := CTypeForInterface(bi, TypeNone)
		if size == 0 {
			printf("struct _%s {};\n", cctype)
		} else {
			printf("struct _%s { uint8_t _data[%d]; };\n", cctype, size)
		}
	}
}

func CForwardDeclarations() string {
	var out bytes.Buffer
	printf = PrinterTo(&out)

	repo := gi.DefaultRepository()
	deps := repo.Dependencies(Config.Namespace)
	for _, dep := range deps {
		depv := strings.Split(dep, "-")
		_, err := repo.Require(depv[0], depv[1], 0)
		if err != nil {
			panic(err)
		}

		for i, n := 0, repo.NumInfo(depv[0]); i < n; i++ {
			CForwardDeclaration10(repo.Info(depv[0], i))
			CForwardDeclaration30(repo.Info(depv[0], i))
		}
	}

	for i, n := 0, repo.NumInfo(Config.Namespace); i < n; i++ {
		CForwardDeclaration10(repo.Info(Config.Namespace, i))
	}
	for i, n := 0, repo.NumInfo(Config.Namespace); i < n; i++ {
		CForwardDeclaration20(repo.Info(Config.Namespace, i))
	}
	for i, n := 0, repo.NumInfo(Config.Namespace); i < n; i++ {
		CForwardDeclaration30(repo.Info(Config.Namespace, i))
	}

	return out.String()
}

func GoUtils() string {
	var out bytes.Buffer
	var gobject string
	var gtype string

	if Config.Namespace == "GObject" {
		gtype = "Type"
		gobject = "Object"
	} else {
		gtype = "gobject.Type"
		gobject = "gobject.Object"
	}

	GoUtilsTemplate.Execute(&out, map[string]interface{}{
		"gobject": gobject,
		"gtype":   gtype,
	})

	return out.String()
}

func GoBindings() string {
	var out bytes.Buffer
	printf = PrinterTo(&out)

	repo := gi.DefaultRepository()
	for i, n := 0, repo.NumInfo(Config.Namespace); i < n; i++ {
		ProcessBaseInfo(repo.Info(Config.Namespace, i))
	}

	return out.String()
}

func ProcessTemplate(tplstr string) {
	tpl := MustTemplate(tplstr)
	tpl.Execute(Config.Sys.Out, map[string]interface{}{
		"CommonIncludes":       CommonIncludes,
		"GType":                GType,
		"CForwardDeclarations": CForwardDeclarations(),
		"GObjectRefUnref":      GObjectRefUnref,
		"GoUtils":              GoUtils(),
		"GoBindings":           GoBindings(),
		"GErrorFree":           GErrorFree,
		"GFree":                GFree,
	})
}

//------------------------------------------------------------------------
//------------------------------------------------------------------------

func ProcessObjectInfo(oi *gi.ObjectInfo) {
	parentStr := "C unsafe.Pointer"
	if parent := oi.Parent(); parent != nil {
		parentStr = ""
		if ns := parent.Namespace(); ns != Config.Namespace {
			parentStr += strings.ToLower(ns) + "."
		}
		parentStr += parent.Name()
	}

	// interface that this class and its subclasses implement
	cprefix := gi.DefaultRepository().CPrefix(Config.Namespace)
	name := oi.Name()
	cgotype := CgoTypeForInterface(gi.ToBaseInfo(oi), TypePointer)
	printf("type %sLike interface {\n", name)
	printf("\tInheritedFrom%s%s() %s\n", cprefix, name, cgotype)
	printf("}\n")

	// the struct itself, uses embedding to emulate inheritance
	printf("type %s struct {\n", name)
	printf("\t%s\n", parentStr)
	printf("}\n")

	// implementation of the above interface
	printf("func (this0 *%s) InheritedFrom%s%s() %s {\n",
		name, cprefix, name, cgotype)
	printf("\treturn (%s)(this0.C)\n", cgotype)
	printf("}\n")

	gtype := "gobject.Type"
	if Config.Namespace == "GObject" {
		gtype = "Type"
	}

	// static type function for closure marshaler
	printf("func (this0 *%s) GetStaticType() %s {\n", name, gtype)
	printf("\treturn %s(C.%s())\n", gtype, oi.TypeInit())
	printf("}\n")

	// interfaces implementation methods
	for i, n := 0, oi.NumInterface(); i < n; i++ {
		ii := oi.Interface(i)
		nm := ii.Name()
		prefix := gi.DefaultRepository().CPrefix(ii.Namespace())
		printf("func (this0 *%s) Implements%s%s() *C.%s%s {\n",
			name, prefix, nm, prefix, nm)
		printf("\treturn (*C.%s%s)(this0.C)\n", prefix, nm)
		printf("}\n")
	}

	object := "gobject.Object"
	if Config.Namespace == "GObject" {
		object = "Object"
	}

	// type casting function
	printf("func To%s(objlike %sLike) (*%s, bool) {\n", name, object, name)
	printf("\tt := ((*%s)(nil)).GetStaticType()\n", name)
	printf("\tc := objlike.InheritedFromGObject()\n")
	printf("\tobj := _GObjectGrabIfType(unsafe.Pointer(c), t)\n")
	printf("\tif obj != nil {\n")
	printf("\t\treturn (*%s)(obj), true\n", name)
	printf("\t}\n")
	printf("\treturn nil, false\n")
	printf("}\n")

	for i, n := 0, oi.NumMethod(); i < n; i++ {
		meth := oi.Method(i)
		if IsMethodBlacklisted(name, meth.Name()) {
			printf("// blacklisted: %s.%s (method)\n", name, meth.Name())
			continue
		}
		ProcessFunctionInfo(meth, gi.ToBaseInfo(oi))
	}
}

func ProcessStructInfo(si *gi.StructInfo) {
	name := si.Name()
	if strings.HasSuffix(name, "Private") {
		return
	}
	if strings.HasSuffix(name, "Class") {
		return
	}
	if strings.HasSuffix(name, "Iface") {
		return
	}
	if strings.HasSuffix(name, "Interface") {
		return
	}

	size := si.Size()

	fullnm := strings.ToLower(si.Namespace()) + "." + si.Name()
	if _, ok := GConfig.Sys.DisguisedTypes[fullnm]; ok {
		size = int(unsafe.Sizeof(unsafe.Pointer(nil)))
	}

	// TODO: ...
	if size != 0 {
		printf("type %s struct { data [%d]byte }\n", name, size)
	} else {
		printf("type %s struct {}\n", name)
	}

	for i, n := 0, si.NumMethod(); i < n; i++ {
		meth := si.Method(i)
		if IsMethodBlacklisted(name, meth.Name()) {
			continue
		}

		ProcessFunctionInfo(meth, gi.ToBaseInfo(si))
	}
}

func ProcessUnionInfo(ui *gi.UnionInfo) {
	name := ui.Name()
	printf("type %s struct {\n", name)
	printf("\t_data [%d]byte\n", ui.Size())
	printf("}\n")

	for i, n := 0, ui.NumMethod(); i < n; i++ {
		meth := ui.Method(i)
		if IsMethodBlacklisted(name, meth.Name()) {
			continue
		}

		ProcessFunctionInfo(meth, gi.ToBaseInfo(ui))
	}
}

func ProcessEnumInfo(ei *gi.EnumInfo) {
	// done
	printf("type %s %s\n", ei.Name(), CgoTypeForTag(ei.StorageType(), TypeNone))
	printf("const (\n")
	for i, n := 0, ei.NumValue(); i < n; i++ {
		val := ei.Value(i)
		printf("\t%s%s %s = %d\n", ei.Name(),
			LowerCaseToCamelCase(val.Name()), ei.Name(), val.Value())
	}
	printf(")\n")
}

func ProcessConstantInfo(ci *gi.ConstantInfo) {
	// done
	name := ci.Name()
	if Config.Namespace == "Gdk" && strings.HasPrefix(name, "KEY_") {
		// KEY_ constants deserve a special treatment
		printf("const Key_%s = %#v\n", name[4:], ci.Value())
		return
	}
	printf("const %s = %#v\n",
		LowerCaseToCamelCase(strings.ToLower(name)),
		ci.Value())
}

func ProcessCallbackInfo(ci *gi.CallableInfo) {
	// TODO: filter out non-closure kinds (we can't handle them anyway)
	printf("type %s func(", ci.Name())
	for i, n := 0, ci.NumArg(); i < n; i++ {
		arg := ci.Arg(i)
		printf("%s %s", arg.Name(), GoType(arg.Type(), TypeNone))
		if i != n-1 {
			printf(", ")
		}
	}
	rt := ci.ReturnType()
	if !rt.IsPointer() && rt.Tag() == gi.TYPE_TAG_VOID {
		printf(")\n")
		return
	}
	printf(") %s\n", GoType(rt, TypeNone))
}

func ProcessFunctionInfo(fi *gi.FunctionInfo, container *gi.BaseInfo) {
	flags := fi.Flags()
	name := fi.Name()

	// --- header
	fb := NewFunctionBuilder(fi)
	printf("func ")
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		// add receiver if it's a method
		printf("(this0 %s) ",
			GoTypeForInterface(container, TypePointer|TypeReturn))
	}
	switch {
	case flags&gi.FUNCTION_IS_CONSTRUCTOR != 0:
		// special names for constructors
		printf("New%s%s(", container.Name(), CtorSuffix(name))
	case flags&gi.FUNCTION_IS_METHOD == 0 && container != nil:
		printf("%s%s(", container.Name(), LowerCaseToCamelCase(name))
	default:
		printf("%s(", LowerCaseToCamelCase(name))
	}
	for i, arg := range fb.Args {
		printf("%s0 %s", arg.ArgInfo.Name(), GoType(arg.TypeInfo, TypeNone))
		if i != len(fb.Args)-1 {
			printf(", ")
		}
	}
	printf(")")
	switch len(fb.Rets) {
	case 0:
		// do nothing if there are not return values
	case 1:
		if flags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
			// override return types for constructors, We can't
			// return generic widget here as C does. Go's type
			// system is stronger.
			printf(" %s",
				GoTypeForInterface(container, TypePointer|TypeReturn))
			break
		}
		if fb.Rets[0].Index == -2 {
			printf(" error")
		} else {
			printf(" %s", GoType(fb.Rets[0].TypeInfo, TypeReturn))
		}
	default:
		printf(" (")
		for i, ret := range fb.Rets {
			if ret.Index == -2 {
				// special error type in Go to represent GError
				printf("error")
				continue
			}
			printf(GoType(ret.TypeInfo, TypeReturn))
			if i != len(fb.Rets)-1 {
				printf(", ")
			}
		}
		printf(")")
	}
	printf(" {\n")

	// --- body stage 1 (Go to C conversions)

	// var declarations
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("\tvar this1 %s\n", CgoTypeForInterface(container, TypePointer))
	}
	for _, arg := range fb.Args {
		printf("\tvar %s1 %s\n", arg.ArgInfo.Name(),
			CgoType(arg.TypeInfo, TypeNone))
		if al := arg.TypeInfo.ArrayLength(); al != -1 {
			arg := fb.OrigArgs[al]
			printf("\tvar %s1 %s\n", arg.Name(),
				CgoType(arg.Type(), TypeNone))
		}
	}

	for _, ret := range fb.Rets {
		if ret.Index == -1 {
			continue
		}

		if ret.Index == -2 {
			printf("\tvar err1 *C.GError\n")
			continue
		}

		if ret.ArgInfo.Direction() == gi.DIRECTION_INOUT {
			continue
		}

		printf("\tvar %s1 %s\n", ret.ArgInfo.Name(),
			CgoType(ret.TypeInfo, TypeNone))
		if al := ret.TypeInfo.ArrayLength(); al != -1 {
			arg := fb.OrigArgs[al]
			printf("\tvar %s1 %s\n", arg.Name(),
				CgoType(arg.Type(), TypeNone))
		}
	}

	// conversions
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		conv := GoToCgoForInterface(container, "this0", "this1", ConvPointer)
		printf("%s", PrintLinesWithIndent(conv))
	}
	for _, arg := range fb.Args {
		nm := arg.ArgInfo.Name()
		conv := GoToCgo(arg.TypeInfo, nm+"0", nm+"1", ConvNone)
		printf("%s", PrintLinesWithIndent(conv))

		// array length
		if len := arg.TypeInfo.ArrayLength(); len != -1 {
			lenarg := fb.OrigArgs[len]
			conv = GoToCgo(lenarg.Type(),
				"len("+nm+"0)", lenarg.Name()+"1", ConvNone)
			printf("\t%s\n", conv)
		}
	}

	// --- body stage 2 (the function call)
	printf("\t")
	if fb.CHasReturnValue() {
		printf("ret1 := ")
	}
	printf("C.%s(", fi.Symbol())
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("this1")
		if len(fb.OrigArgs) > 0 {
			printf(", ")
		}
	}
	for i, oarg := range fb.OrigArgs {
		var arg string
		dir := oarg.Direction()
		if dir == gi.DIRECTION_INOUT || dir == gi.DIRECTION_OUT {
			arg = fmt.Sprintf("&%s1", oarg.Name())
		} else {
			arg = fmt.Sprintf("%s1", oarg.Name())
		}

		printf("%s", arg)

		if i != len(fb.OrigArgs)-1 {
			printf(", ")
		}
	}
	if flags&gi.FUNCTION_THROWS != 0 {
		printf(", &err1")
	}
	printf(")\n")

	// --- body stage 3 (C to Go conversions)

	// var declarations
	for _, ret := range fb.Rets {
		switch ret.Index {
		case -1:
			if flags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
				printf("\tvar ret2 %s\n",
					GoTypeForInterface(container, TypePointer|TypeReturn))
			} else {
				printf("\tvar ret2 %s\n", GoType(ret.TypeInfo, TypeReturn))
			}
		case -2:
			printf("\tvar err2 error\n")
		default:
			printf("\tvar %s2 %s\n", ret.ArgInfo.Name(),
				GoType(ret.TypeInfo, TypeReturn))
		}
	}

	// conversions
	for _, ret := range fb.Rets {
		if ret.Index == -2 {
			printf("\tif err1 != nil {\n")
			printf("\t\terr2 = errors.New(C.GoString(((*_GError)(unsafe.Pointer(err1))).message))\n")
			printf("\t\tC.g_error_free(err1)\n")
			printf("\t}\n")
			continue
		}

		var nm string
		if ret.Index == -1 {
			nm = "ret"
		} else {
			nm = ret.ArgInfo.Name()
		}

		// array length
		if len := ret.TypeInfo.ArrayLength(); len != -1 {
			lenarg := fb.OrigArgs[len]
			printf("\t%s2 = make(%s, %s)\n",
				nm, GoType(ret.TypeInfo, TypeReturn),
				lenarg.Name()+"1")
		}

		var ownership gi.Transfer
		if ret.Index == -1 {
			ownership = fi.CallerOwns()
		} else {
			ownership = ret.ArgInfo.OwnershipTransfer()
			if ret.ArgInfo.Direction() == gi.DIRECTION_INOUT {
				// not sure if it's true in all cases, but so far
				ownership = gi.TRANSFER_NOTHING
			}
		}
		var conv string
		if flags&gi.FUNCTION_IS_CONSTRUCTOR != 0 && ret.Index == -1 {
			conv = CgoToGoForInterface(container, "ret1", "ret2",
				ConvPointer|OwnershipToConvFlags(ownership))
		} else {
			conv = CgoToGo(ret.TypeInfo, nm+"1", nm+"2",
				OwnershipToConvFlags(ownership))
		}
		printf("%s", PrintLinesWithIndent(conv))
	}

	// --- body stage 4 (return)
	if len(fb.Rets) == 0 {
		printf("}\n")
		return
	}

	printf("\treturn ")
	for i, ret := range fb.Rets {
		var nm string
		switch ret.Index {
		case -1:
			nm = "ret2"
		case -2:
			nm = "err2"
		default:
			nm = ret.ArgInfo.Name() + "2"
		}
		if i != 0 {
			printf(", ")
		}
		printf(nm)
	}

	printf("\n}\n")
}

func ProcessInterfaceInfo(ii *gi.InterfaceInfo) {
	nm := ii.Name()
	prefix := gi.DefaultRepository().CPrefix(ii.Namespace())

	printf("type %s interface {\n", nm)
	printf("\tImplements%s%s() *C.%s%s\n", prefix, nm,
		prefix, nm)
	printf("}\n")

	// plus dummy struct that implements that interface (for return values)
	printf("type %sDummy struct {\n", nm)
	printf("\tC unsafe.Pointer\n")
	printf("}\n")

	printf("func (this0 *%sDummy) Implements%s%s() *C.%s%s {\n",
		nm, prefix, nm, prefix, nm)
	printf("\treturn (*C.%s%s)(this0.C)\n", prefix, nm)
	printf("}\n")
}

func ProcessBaseInfo(bi *gi.BaseInfo) {
	switch bi.Type() {
	case gi.INFO_TYPE_UNION:
		if IsBlacklisted("unions", bi.Name()) {
			goto blacklisted
		}
		ProcessUnionInfo(gi.ToUnionInfo(bi))
	case gi.INFO_TYPE_STRUCT:
		if IsBlacklisted("structs", bi.Name()) {
			goto blacklisted
		}
		ProcessStructInfo(gi.ToStructInfo(bi))
	case gi.INFO_TYPE_ENUM, gi.INFO_TYPE_FLAGS:
		if IsBlacklisted("enums", bi.Name()) {
			goto blacklisted
		}
		ProcessEnumInfo(gi.ToEnumInfo(bi))
	case gi.INFO_TYPE_CONSTANT:
		if IsBlacklisted("constants", bi.Name()) {
			goto blacklisted
		}
		ProcessConstantInfo(gi.ToConstantInfo(bi))
	case gi.INFO_TYPE_CALLBACK:
		if IsBlacklisted("callbacks", bi.Name()) {
			goto blacklisted
		}
		ProcessCallbackInfo(gi.ToCallableInfo(bi))
	case gi.INFO_TYPE_FUNCTION:
		if IsBlacklisted("functions", bi.Name()) {
			goto blacklisted
		}
		ProcessFunctionInfo(gi.ToFunctionInfo(bi), nil)
	case gi.INFO_TYPE_INTERFACE:
		if IsBlacklisted("interfaces", bi.Name()) {
			goto blacklisted
		}
		ProcessInterfaceInfo(gi.ToInterfaceInfo(bi))
	case gi.INFO_TYPE_OBJECT:
		if IsBlacklisted("objects", bi.Name()) {
			goto blacklisted
		}
		ProcessObjectInfo(gi.ToObjectInfo(bi))
	default:
		printf("// TODO: %s (%s)\n", bi.Name(), bi.Type())
	}
	return

blacklisted:
	printf("// blacklisted: %s (%s)\n", bi.Name(), bi.Type())
}
