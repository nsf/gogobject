package main

import (
	"bytes"
	"fmt"
	"gobject/gi"
	"strings"
)

var declared_c_funcs = make(map[string]bool)

func CFuncForwardDeclaration(fi *gi.FunctionInfo, container *gi.BaseInfo) {
	symbol := fi.Symbol()
	if _, declared := declared_c_funcs[symbol]; declared {
		return
	}
	declared_c_funcs[symbol] = true

	flags := fi.Flags()
	narg := fi.NumArg()
	if flags&gi.FUNCTION_THROWS != 0 {
		narg++
	}

	printf("extern %s %s(", CType(fi.ReturnType(), TypeNone), symbol)
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("%s", CTypeForInterface(container, TypePointer))
		if narg > 0 {
			printf(", ")
		}
	}

	var closure_scope gi.ScopeType = gi.SCOPE_TYPE_INVALID
	var closure_userdata int = -1
	var closure_destroy int = -1
	var closure_callback int = -1

	for i, n := 0, narg; i < n; i++ {
		if i != 0 {
			printf(", ")
		}

		if i == n-1 && flags&gi.FUNCTION_THROWS != 0 {
			printf("GError**")
			continue
		}

		arg := fi.Arg(i)
		t := arg.Type()

		if uarg := arg.Closure(); uarg != -1 {
			if t.Tag() == gi.TYPE_TAG_INTERFACE {
				if t.Interface().Type() == gi.INFO_TYPE_CALLBACK {
					if darg := arg.Destroy(); darg != -1 {
						closure_destroy = darg
					}
					closure_userdata = uarg
					closure_callback = i
					closure_scope = arg.Scope()
				}
			}
		}

		printf("%s", CType(t, TypeNone))

		switch arg.Direction() {
		case gi.DIRECTION_INOUT, gi.DIRECTION_OUT:
			printf("*")
		}
	}
	printf(");\n")

	if closure_scope == gi.SCOPE_TYPE_INVALID {
		return
	}
	// in case if the function takes callback, generate appropriate wrappers
	// for Go (gc compiler mainly)
	printf("#pragma GCC diagnostic ignored \"-Wunused-function\"\n")
	printf("static %s _%s(", CType(fi.ReturnType(), TypeNone), symbol)
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("%s this", CTypeForInterface(container, TypePointer))
		if narg > 0 {
			printf(", ")
		}
	}

	for i, n := 0, narg; i < n; i++ {
		if i == closure_userdata || i == closure_destroy {
			// skip userdata and destroy func in the wrapper
			continue
		}

		if i != 0 {
			printf(", ")
		}

		if i == closure_callback {
			// replace callback argument with Go function pointer
			// and optional unique id (if scope is not CALL)
			printf("void* gofunc")
			continue
		}

		if i == n-1 && flags&gi.FUNCTION_THROWS != 0 {
			printf("GError** arg%d", i)
			continue
		}

		arg := fi.Arg(i)
		printf("%s", CType(arg.Type(), TypeNone))

		switch arg.Direction() {
		case gi.DIRECTION_INOUT, gi.DIRECTION_OUT:
			printf("*")
		}
		printf(" arg%d", i)
	}
	printf(") {\n")

	// body
	printf("\t")
	if ret := fi.ReturnType(); !(ret.Tag() == gi.TYPE_TAG_VOID && !ret.IsPointer()) {
		printf("return ")
	}
	printf("%s(", symbol)
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("this")
		if narg > 0 {
			printf(", ")
		}
	}
	for i, n := 0, narg; i < n; i++ {
		arg := fi.Arg(i)
		t := arg.Type()

		if i != 0 {
			printf(", ")
		}

		switch i {
		case closure_userdata:
			printf("gofunc")
		case closure_destroy:
			printf("_c_callback_cleanup")
		case closure_callback:
			printf("_%s_c_wrapper", CType(t, TypeNone))
			if closure_scope == gi.SCOPE_TYPE_ASYNC {
				printf("_once")
			}
		default:
			printf("arg%d", i)
		}
	}
	printf(");\n")
	printf("}\n")
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
		fullnm := strings.ToLower(bi.Namespace()) + "." + bi.Name()
		cctype := CTypeForInterface(bi, TypeNone)
		if _, ok := GConfig.Sys.DisguisedTypes[fullnm]; ok {
			printf("typedef void *%s;\n", cctype)
		} else {
			printf("typedef struct _%s %s;\n", cctype, cctype)
		}
	case gi.INFO_TYPE_CALLBACK:
		ctype := CTypeForInterface(bi, TypeNone)
		// type doesn't matter here, it's just a pointer after all
		printf("typedef void* %s;\n", ctype)
		// also generate wrapper declarations
		printf("extern void _%s_c_wrapper();\n", ctype)
		printf("extern void _%s_c_wrapper_once();\n", ctype)
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
	case gi.INFO_TYPE_INTERFACE:
		ii := gi.ToInterfaceInfo(bi)
		for i, n := 0, ii.NumMethod(); i < n; i++ {
			meth := ii.Method(i)
			CFuncForwardDeclaration(meth, bi)
		}
		printf("extern GType %s();\n", ii.TypeInit())
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
			return
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
		"namespace": Config.Namespace,
	})

	return out.String()
}

func CUtils() string {
	var out bytes.Buffer
	CUtilsTemplate.Execute(&out, map[string]interface{}{
		"namespace": Config.Namespace,
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
		"CUtils":               CUtils(),
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
	parent := "C unsafe.Pointer"
	if p := oi.Parent(); p != nil {
		parent = ""
		if ns := p.Namespace(); ns != Config.Namespace {
			parent += strings.ToLower(ns) + "."
		}
		parent += p.Name()
	}

	// interface that this class and its subclasses implement
	cprefix := gi.DefaultRepository().CPrefix(Config.Namespace)
	name := oi.Name()
	cgotype := CgoTypeForInterface(gi.ToBaseInfo(oi), TypePointer)

	gtype := "gobject.Type"
	if Config.Namespace == "GObject" {
		gtype = "Type"
	}

	gobject := "gobject.Object"
	if Config.Namespace == "GObject" {
		gobject = "Object"
	}

	var interfaces bytes.Buffer
	for i, n := 0, oi.NumInterface(); i < n; i++ {
		ii := oi.Interface(i)
		name := ii.Name()
		ns := ii.Namespace()
		if i != 0 {
			interfaces.WriteString("\n\t")
		}
		if ns != Config.Namespace {
			fmt.Fprintf(&interfaces, "%s.", strings.ToLower(ns))
		}
		fmt.Fprintf(&interfaces, "%sImpl", name)
	}

	printf("%s\n", ExecuteTemplate(ObjectTemplate, map[string]string{
		"name":       name,
		"cprefix":    cprefix,
		"cgotype":    cgotype,
		"parent":     parent,
		"gtype":      gtype,
		"typeinit":   oi.TypeInit(),
		"gobject":    gobject,
		"interfaces": interfaces.String(),
	}))

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
		size = -1
	}

	switch size {
	case -1:
		printf("type %s struct { Pointer unsafe.Pointer }\n", name)
	case 0:
		printf("type %s struct {}\n", name)
	default:
		printf("type %s struct { data [%d]byte }\n", name, size)
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
	// list of args
	var args []*gi.ArgInfo
	for i, n := 0, ci.NumArg(); i < n; i++ {
		arg := ci.Arg(i)
		args = append(args, arg)
	}

	userdata := -1
	for i, arg := range args {
		if arg.Closure() != -1 {
			userdata = i
			break
		}

		// treat any void* as userdata O_o
		t := arg.Type()
		if t.Tag() == gi.TYPE_TAG_VOID && t.IsPointer() {
			userdata = i
			break
		}
	}
	if userdata == -1 {
		printf("// blacklisted (no userdata): ")
	}

	name := ci.Name()
	printf("type %s func(", name)
	for i, ri, n := 0, 0, len(args); i < n; i++ {
		if i == userdata {
			continue
		}

		// I use here TypeReturn because for closures it's inverted
		// C code calls closure and it has to be concrete and Go code
		// returns stuff to C (like calling a C function)
		arg := args[i]
		if ri != 0 {
			printf(", ")
		}
		printf("%s %s", arg.Name(), GoType(arg.Type(), TypeReturn))
		ri++
	}
	rt := ci.ReturnType()
	if !rt.IsPointer() && rt.Tag() == gi.TYPE_TAG_VOID {
		printf(")\n")
	} else {
		printf(") %s\n", GoType(rt, TypeNone))
	}
	if userdata == -1 {
		return
	}

	// now we need to generate two wrappers, it's a bit tricky due to cgo
	// ugliness.
	// 1. Function signature should consist of basic types and unsafe.Pointer
	//    for any pointer type, cgo cannot export other kinds of functions.
	// 2. Convert these types to C.* ones.
	// 3. Convert C.* types to Go values.
	// 4. Call Go callback (function pointer is in the userdata).
	// 5. Convert all returned Go values to C.* types and then to basic cgo
	//    friendly types.

	// signature
	ctype := CTypeForInterface(gi.ToBaseInfo(ci), TypeNone)
	printf("//export _%s_c_wrapper\n", ctype)
	printf("func _%s_c_wrapper(", ctype)
	for i, arg := range args {
		if i != 0 {
			printf(", ")
		}
		printf("%s0 %s", arg.Name(), SimpleCgoType(arg.Type(), TypeNone))
	}
	printf(") ")
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("%s ", SimpleCgoType(ret, TypeNone))
	}

	printf("{\n")
	// --- body stage 1 (C to Go conversions)

	// var declarations
	for i, arg := range args {
		gotype := GoType(arg.Type(), TypeReturn)
		if i == userdata {
			gotype = name
		}
		printf("\tvar %s1 %s\n", arg.Name(), gotype)
	}

	// conversions
	for i, arg := range args {
		t := arg.Type()
		aname := arg.Name()
		if i == userdata {
			printf("\t%s1 = *(*%s)(%s0)\n", aname, name, aname)
			continue
		}
		ownership := OwnershipToConvFlags(arg.OwnershipTransfer())
		conv := SimpleCgoToGo(t, aname+"0", aname+"1", ownership)
		printf("%s", PrintLinesWithIndent(conv))
	}

	// --- body stage 2 (the callback call)
	printf("\t")
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("ret1 := ")
	}
	printf("%s1(", args[userdata].Name())
	for i, ri, n := 0, 0, len(args); i < n; i++ {
		if i == userdata {
			continue
		}
		if ri != 0 {
			printf(", ")
		}
		printf("%s1", args[i].Name())
		ri++
	}
	printf(")\n")

	// --- body stage 3 (return value)
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("\tvar ret2 %s\n", CgoType(ret, TypeNone))
		ownership := OwnershipToConvFlags(ci.CallerOwns())
		conv := GoToCgo(ret, "ret1", "ret2", ownership)
		printf("%s", PrintLinesWithIndent(conv))
		printf("\treturn (%s)(ret2)\n", SimpleCgoType(ret, TypeNone))
	}
	printf("}\n")

	// and finally add "_once" wrapper
	printf("//export _%s_c_wrapper_once\n", ctype)
	printf("func _%s_c_wrapper_once(", ctype)
	for i, arg := range args {
		if i != 0 {
			printf(", ")
		}
		printf("%s0 %s", arg.Name(), SimpleCgoType(arg.Type(), TypeNone))
	}
	printf(") ")
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("%s ", SimpleCgoType(ret, TypeNone))
	}
	printf("{\n\t")
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("ret := ")
	}
	printf("_%s_c_wrapper(", ctype)
	for i, arg := range args {
		if i != 0 {
			printf(", ")
		}
		printf("%s0", arg.Name())
	}
	printf(")\n")
	printf("\t_%s_go_callback_cleanup(%s0)\n", Config.Namespace,
		args[userdata].Name())
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("\treturn ret\n")
	}
	printf("}\n")
}

func ProcessFunctionInfo(fi *gi.FunctionInfo, container *gi.BaseInfo) {
	var fullnm string
	flags := fi.Flags()
	name := fi.Name()

	// --- header
	fb := NewFunctionBuilder(fi)
	printf("func ")
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		// add receiver if it's a method
		printf("(this0 %s) ",
			GoTypeForInterface(container, TypePointer|TypeReceiver))
		fullnm = container.Name() + "."
	}
	switch {
	case flags&gi.FUNCTION_IS_CONSTRUCTOR != 0:
		// special names for constructors
		name = fmt.Sprintf("New%s%s", container.Name(), CtorSuffix(name))
	case flags&gi.FUNCTION_IS_METHOD == 0 && container != nil:
		name = fmt.Sprintf("%s%s", container.Name(), LowerCaseToCamelCase(name))
	default:
		name = fmt.Sprintf("%s", LowerCaseToCamelCase(name))
	}
	fullnm += name
	name = Rename(fullnm, name)
	printf("%s(", name)
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

		// register callback in the global map
		if arg.TypeInfo.Tag() == gi.TYPE_TAG_INTERFACE {
			bi := arg.TypeInfo.Interface()
			if bi.Type() == gi.INFO_TYPE_CALLBACK {
				if arg.ArgInfo.Scope() != gi.SCOPE_TYPE_CALL {
					printf("\t_cbcache[%s1] = true\n", nm)
				}
			}
		}

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
	if fb.HasReturnValue() {
		printf("ret1 := ")
	}

	userdata, destroy, scope := fb.HasClosureArgument()
	printf("C.")
	if scope != gi.SCOPE_TYPE_INVALID {
		printf("_")
	}
	printf("%s(", fi.Symbol())
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("this1")
		if len(fb.OrigArgs) > 0 {
			printf(", ")
		}
	}
	for i, ri, n := 0, 0, len(fb.OrigArgs); i < n; i++ {
		if i == userdata || i == destroy {
			continue
		}

		oarg := fb.OrigArgs[i]

		var arg string
		dir := oarg.Direction()
		if dir == gi.DIRECTION_INOUT || dir == gi.DIRECTION_OUT {
			arg = fmt.Sprintf("&%s1", oarg.Name())
		} else {
			arg = fmt.Sprintf("%s1", oarg.Name())
		}

		if ri != 0 {
			printf(", ")
		}
		printf("%s", arg)
		ri++
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
	name := ii.Name()
	cprefix := gi.DefaultRepository().CPrefix(ii.Namespace())
	cgotype := CgoTypeForInterface(gi.ToBaseInfo(ii), TypePointer)

	gtype := "gobject.Type"
	if Config.Namespace == "GObject" {
		gtype = "Type"
	}

	gobject := "gobject.Object"
	if Config.Namespace == "GObject" {
		gobject = "Object"
	}

	printf("%s\n", ExecuteTemplate(InterfaceTemplate, map[string]string{
		"name":     name,
		"cprefix":  cprefix,
		"cgotype":  cgotype,
		"gtype":    gtype,
		"typeinit": ii.TypeInit(),
		"gobject":  gobject,
	}))

	for i, n := 0, ii.NumMethod(); i < n; i++ {
		meth := ii.Method(i)
		if IsMethodBlacklisted(name, meth.Name()) {
			printf("// blacklisted: %s.%s (method)\n", name, meth.Name())
			continue
		}
		ProcessFunctionInfo(meth, gi.ToBaseInfo(ii))
	}
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
