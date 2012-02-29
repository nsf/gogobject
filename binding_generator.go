package main

import (
	"bytes"
	"fmt"
	"gobject/gi"
	"strings"
)

var declared_c_funcs = make(map[string]bool)

func c_func_forward_declaration(fi *gi.FunctionInfo, container *gi.BaseInfo) {
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

	printf("extern %s %s(", c_type(fi.ReturnType(), type_none), symbol)
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("%s", c_type_for_interface(container, type_pointer))
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

		printf("%s", c_type(t, type_none))

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
	printf("static %s _%s(", c_type(fi.ReturnType(), type_none), symbol)
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("%s this", c_type_for_interface(container, type_pointer))
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
		printf("%s", c_type(arg.Type(), type_none))

		switch arg.Direction() {
		case gi.DIRECTION_INOUT, gi.DIRECTION_OUT:
			printf("*")
		}
		printf(" arg%d", i)
	}
	printf(") {\n")

	// body
	defcall := func(argprint func(i int, t *gi.TypeInfo)) {
		printf("\t\t")
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

			argprint(i, t)
		}
		printf(");\n")
	}
	printf("\tif (gofunc) {\n")
	defcall(func(i int, t *gi.TypeInfo) {
		switch i {
		case closure_userdata:
			printf("gofunc")
		case closure_destroy:
			printf("_c_callback_cleanup")
		case closure_callback:
			printf("_%s_c_wrapper", c_type(t, type_none))
			if closure_scope == gi.SCOPE_TYPE_ASYNC {
				printf("_once")
			}
		default:
			printf("arg%d", i)
		}
	})
	printf("\t} else {\n")
	defcall(func(i int, t *gi.TypeInfo) {
		switch i {
		case closure_userdata, closure_destroy, closure_callback:
			printf("0")
		default:
			printf("arg%d", i)
		}
	})
	printf("\t}\n")
	printf("}\n")
}


// generating forward C declarations properly:
// 10 - various typedefs
// 20 - functions and methods
// 30 - struct definitions

func c_forward_declaration10(bi *gi.BaseInfo) {
	switch bi.Type() {
	case gi.INFO_TYPE_OBJECT:
		cctype := c_type_for_interface(bi, type_none)
		printf("typedef struct _%s %s;\n", cctype, cctype)
	case gi.INFO_TYPE_FLAGS, gi.INFO_TYPE_ENUM:
		ei := gi.ToEnumInfo(bi)
		printf("typedef %s %s;\n", c_type_for_tag(ei.StorageType(), type_none),
			c_type_for_interface(bi, type_none))
	case gi.INFO_TYPE_STRUCT, gi.INFO_TYPE_INTERFACE, gi.INFO_TYPE_UNION:
		fullnm := strings.ToLower(bi.Namespace()) + "." + bi.Name()
		cctype := c_type_for_interface(bi, type_none)
		if _, ok := g_commonconfig.sys.disguised_types[fullnm]; ok {
			printf("typedef void *%s;\n", cctype)
		} else {
			printf("typedef struct _%s %s;\n", cctype, cctype)
		}
	case gi.INFO_TYPE_CALLBACK:
		ctype := c_type_for_interface(bi, type_none)
		// type doesn't matter here, it's just a pointer after all
		printf("typedef void* %s;\n", ctype)
		// also generate wrapper declarations
		printf("extern void _%s_c_wrapper();\n", ctype)
		printf("extern void _%s_c_wrapper_once();\n", ctype)
	}
}

func c_forward_declaration20(bi *gi.BaseInfo) {
	switch bi.Type() {
	case gi.INFO_TYPE_FUNCTION:
		fi := gi.ToFunctionInfo(bi)
		c_func_forward_declaration(fi, nil)
	case gi.INFO_TYPE_OBJECT:
		oi := gi.ToObjectInfo(bi)
		for i, n := 0, oi.NumMethod(); i < n; i++ {
			meth := oi.Method(i)
			c_func_forward_declaration(meth, bi)
		}
		printf("extern GType %s();\n", oi.TypeInit())
	case gi.INFO_TYPE_INTERFACE:
		ii := gi.ToInterfaceInfo(bi)
		for i, n := 0, ii.NumMethod(); i < n; i++ {
			meth := ii.Method(i)
			c_func_forward_declaration(meth, bi)
		}
		printf("extern GType %s();\n", ii.TypeInit())
	case gi.INFO_TYPE_STRUCT:
		si := gi.ToStructInfo(bi)
		for i, n := 0, si.NumMethod(); i < n; i++ {
			meth := si.Method(i)
			c_func_forward_declaration(meth, bi)
		}
	case gi.INFO_TYPE_UNION:
		ui := gi.ToUnionInfo(bi)
		for i, n := 0, ui.NumMethod(); i < n; i++ {
			meth := ui.Method(i)
			c_func_forward_declaration(meth, bi)
		}
	}
}

func c_forward_declaration30(bi *gi.BaseInfo) {
	fullnm := strings.ToLower(bi.Namespace()) + "." + bi.Name()

	switch bi.Type() {
	case gi.INFO_TYPE_STRUCT:
		si := gi.ToStructInfo(bi)
		size := si.Size()
		if _, ok := g_commonconfig.sys.disguised_types[fullnm]; ok {
			return
		}
		cctype := c_type_for_interface(bi, type_none)
		if size == 0 {
			printf("struct _%s {};\n", cctype)
		} else {
			printf("struct _%s { uint8_t _data[%d]; };\n", cctype, size)
		}
	case gi.INFO_TYPE_UNION:
		ui := gi.ToUnionInfo(bi)
		size := ui.Size()
		cctype := c_type_for_interface(bi, type_none)
		if size == 0 {
			printf("struct _%s {};\n", cctype)
		} else {
			printf("struct _%s { uint8_t _data[%d]; };\n", cctype, size)
		}
	}
}

func c_forward_declarations() string {
	var out bytes.Buffer
	printf = printer_to(&out)

	repo := gi.DefaultRepository()
	deps := repo.Dependencies(g_config.Namespace)
	for _, dep := range deps {
		depv := strings.Split(dep, "-")
		_, err := repo.Require(depv[0], depv[1], 0)
		if err != nil {
			panic(err)
		}

		for i, n := 0, repo.NumInfo(depv[0]); i < n; i++ {
			c_forward_declaration10(repo.Info(depv[0], i))
			c_forward_declaration30(repo.Info(depv[0], i))
		}
	}

	for i, n := 0, repo.NumInfo(g_config.Namespace); i < n; i++ {
		c_forward_declaration10(repo.Info(g_config.Namespace, i))
	}
	for i, n := 0, repo.NumInfo(g_config.Namespace); i < n; i++ {
		c_forward_declaration20(repo.Info(g_config.Namespace, i))
	}
	for i, n := 0, repo.NumInfo(g_config.Namespace); i < n; i++ {
		c_forward_declaration30(repo.Info(g_config.Namespace, i))
	}

	return out.String()
}

func go_utils(cb bool) string {
	var out bytes.Buffer

	go_utils_template.Execute(&out, map[string]interface{}{
		"gobjectns": g_config.sys.gns,
		"namespace": g_config.Namespace,
		"nocallbacks": !cb,
	})

	return out.String()
}

func c_utils() string {
	var out bytes.Buffer
	c_utils_template.Execute(&out, map[string]interface{}{
		"namespace": g_config.Namespace,
	})

	return out.String()
}

func go_bindings() string {
	var out bytes.Buffer
	printf = printer_to(&out)

	repo := gi.DefaultRepository()
	for i, n := 0, repo.NumInfo(g_config.Namespace); i < n; i++ {
		process_base_info(repo.Info(g_config.Namespace, i))
	}

	return out.String()
}

func process_template(tplstr string) {
	tpl := must_template(tplstr)
	tpl.Execute(g_config.sys.out, map[string]interface{}{
		"CommonIncludes":       common_includes,
		"GType":                g_type,
		"CUtils":               c_utils(),
		"CForwardDeclarations": c_forward_declarations(),
		"GObjectRefUnref":      g_object_ref_unref,
		"GoUtils":              go_utils(true),
		"GoUtilsNoCB":		go_utils(false),
		"GoBindings":           go_bindings(),
		"GErrorFree":           g_error_free,
		"GFree":                g_free,
	})
}

//------------------------------------------------------------------------
//------------------------------------------------------------------------

func process_object_info(oi *gi.ObjectInfo) {
	parent := "C unsafe.Pointer"
	parentlike := ""
	if p := oi.Parent(); p != nil {
		parent = ""
		if ns := p.Namespace(); ns != g_config.Namespace {
			parent += strings.ToLower(ns) + "."
		}
		parent += p.Name()
		parentlike = parent + "Like"
	}

	// interface that this class and its subclasses implement
	cprefix := gi.DefaultRepository().CPrefix(g_config.Namespace)
	name := oi.Name()
	cgotype := cgo_type_for_interface(gi.ToBaseInfo(oi), type_pointer)

	var interfaces bytes.Buffer
	for i, n := 0, oi.NumInterface(); i < n; i++ {
		ii := oi.Interface(i)
		name := ii.Name()
		ns := ii.Namespace()
		if i != 0 {
			interfaces.WriteString("\n\t")
		}
		if ns != g_config.Namespace {
			fmt.Fprintf(&interfaces, "%s.", strings.ToLower(ns))
		}
		fmt.Fprintf(&interfaces, "%sImpl", name)
	}

	printf("%s\n", execute_template(object_template, map[string]string{
		"name":       name,
		"cprefix":    cprefix,
		"cgotype":    cgotype,
		"parent":     parent,
		"parentlike": parentlike,
		"typeinit":   oi.TypeInit(),
		"gobjectns":  g_config.sys.gns,
		"interfaces": interfaces.String(),
	}))

	for i, n := 0, oi.NumMethod(); i < n; i++ {
		meth := oi.Method(i)
		if g_config.is_method_blacklisted(name, meth.Name()) {
			printf("// blacklisted: %s.%s (method)\n", name, meth.Name())
			continue
		}
		process_function_info(meth, gi.ToBaseInfo(oi))
	}
}

func process_struct_info(si *gi.StructInfo) {
	name := si.Name()
	size := si.Size()

	if si.IsGTypeStruct() {
		return
	}
	if strings.HasSuffix(name, "Private") {
		return
	}

	fullnm := strings.ToLower(si.Namespace()) + "." + name
	if _, ok := g_commonconfig.sys.disguised_types[fullnm]; ok {
		size = -1
	}

	if !g_config.is_blacklisted("structdefs", name) {
		switch size {
		case -1:
			printf("type %s struct { Pointer unsafe.Pointer }\n", name)
		case 0:
			printf("type %s struct {}\n", name)
		default:
			printf("type %s struct {\n", name)
			offset := 0
			for i, n := 0, si.NumField(); i < n; i++ {
				field := si.Field(i)
				fo := field.Offset()
				ft := field.Type()
				nm := field.Name()
				if fo != offset {
					pad := fo - offset
					printf("\t_ [%d]byte\n", pad)
					offset += pad
				}
				if type_needs_wrapper(ft) {
					printf("\t%s0 %s\n", nm, cgo_type(ft, type_exact))
				} else {
					printf("\t%s %s\n", lower_case_to_camel_case(nm),
						go_type(ft, type_exact))
				}
				offset += type_size(ft, type_exact)
			}
			if size != offset {
				printf("\t_ [%d]byte\n", size - offset)
			}
			printf("}\n")
			//printf("type %s struct { data [%d]byte }\n", name, size)
		}

		// for each field that needs a wrapper, generate it
		for i, n := 0, si.NumField(); i < n; i++ {
			field := si.Field(i)
			ft := field.Type()
			nm := field.Name()

			if !type_needs_wrapper(ft) {
				continue
			}

			gotype := go_type(ft, type_return)
			printf("func (this0 *%s) %s() %s {\n",
				name, lower_case_to_camel_case(nm), gotype)
			printf("\tvar %s1 %s\n", nm, gotype)
			conv := cgo_to_go(ft, "this0."+nm+"0", nm+"1",
				conv_own_none)
			printf("%s", print_lines_with_indent(conv))
			printf("\treturn %s1\n", nm)
			printf("}\n")
		}
	}

	for i, n := 0, si.NumMethod(); i < n; i++ {
		meth := si.Method(i)
		if g_config.is_method_blacklisted(name, meth.Name()) {
			continue
		}

		process_function_info(meth, gi.ToBaseInfo(si))
	}
}

func process_union_info(ui *gi.UnionInfo) {
	name := ui.Name()
	printf("type %s struct {\n", name)
	printf("\t_data [%d]byte\n", ui.Size())
	printf("}\n")

	for i, n := 0, ui.NumMethod(); i < n; i++ {
		meth := ui.Method(i)
		if g_config.is_method_blacklisted(name, meth.Name()) {
			continue
		}

		process_function_info(meth, gi.ToBaseInfo(ui))
	}
}

func process_enum_info(ei *gi.EnumInfo) {
	// done
	printf("type %s %s\n", ei.Name(), cgo_type_for_tag(ei.StorageType(), type_none))
	printf("const (\n")
	for i, n := 0, ei.NumValue(); i < n; i++ {
		val := ei.Value(i)
		printf("\t%s%s %s = %d\n", ei.Name(),
			lower_case_to_camel_case(val.Name()), ei.Name(), val.Value())
	}
	printf(")\n")
}

func process_constant_info(ci *gi.ConstantInfo) {
	// done
	name := ci.Name()
	if g_config.Namespace == "Gdk" && strings.HasPrefix(name, "KEY_") {
		// KEY_ constants deserve a special treatment
		printf("const Key_%s = %#v\n", name[4:], ci.Value())
		return
	}
	printf("const %s = %#v\n",
		lower_case_to_camel_case(strings.ToLower(name)),
		ci.Value())
}

func process_callback_info(ci *gi.CallableInfo) {
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
		printf("%s %s", arg.Name(), go_type(arg.Type(), type_return))
		ri++
	}
	rt := ci.ReturnType()
	if !rt.IsPointer() && rt.Tag() == gi.TYPE_TAG_VOID {
		printf(")\n")
	} else {
		printf(") %s\n", go_type(rt, type_none))
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
	ctype := c_type_for_interface(gi.ToBaseInfo(ci), type_none)
	printf("//export _%s_c_wrapper\n", ctype)
	printf("func _%s_c_wrapper(", ctype)
	for i, arg := range args {
		if i != 0 {
			printf(", ")
		}
		printf("%s0 %s", arg.Name(), simple_cgo_type(arg.Type(), type_none))
	}
	printf(") ")
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("%s ", simple_cgo_type(ret, type_none))
	}

	printf("{\n")
	// --- body stage 1 (C to Go conversions)

	// var declarations
	for i, arg := range args {
		gotype := go_type(arg.Type(), type_return)
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
		ownership := ownership_to_conv_flags(arg.OwnershipTransfer())
		conv := simple_cgo_to_go(t, aname+"0", aname+"1", ownership)
		printf("%s", print_lines_with_indent(conv))
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
		printf("\tvar ret2 %s\n", cgo_type(ret, type_none))
		ownership := ownership_to_conv_flags(ci.CallerOwns())
		conv := go_to_cgo(ret, "ret1", "ret2", ownership)
		printf("%s", print_lines_with_indent(conv))
		printf("\treturn (%s)(ret2)\n", simple_cgo_type(ret, type_none))
	}
	printf("}\n")

	// and finally add "_once" wrapper
	printf("//export _%s_c_wrapper_once\n", ctype)
	printf("func _%s_c_wrapper_once(", ctype)
	for i, arg := range args {
		if i != 0 {
			printf(", ")
		}
		printf("%s0 %s", arg.Name(), simple_cgo_type(arg.Type(), type_none))
	}
	printf(") ")
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("%s ", simple_cgo_type(ret, type_none))
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
	printf("\t%sHolder.Release(%s0)\n", g_config.sys.gns,
		args[userdata].Name())
	if ret := ci.ReturnType(); ret != nil && ret.Tag() != gi.TYPE_TAG_VOID {
		printf("\treturn ret\n")
	}
	printf("}\n")
}

func process_function_info(fi *gi.FunctionInfo, container *gi.BaseInfo) {
	var fullnm string
	flags := fi.Flags()
	name := fi.Name()

	// --- header
	fb := new_function_builder(fi)
	printf("func ")
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		// add receiver if it's a method
		printf("(this0 %s) ",
			go_type_for_interface(container, type_pointer|type_receiver))
		fullnm = container.Name() + "."
	}
	switch {
	case flags&gi.FUNCTION_IS_CONSTRUCTOR != 0:
		// special names for constructors
		name = fmt.Sprintf("New%s%s", container.Name(), ctor_suffix(name))
	case flags&gi.FUNCTION_IS_METHOD == 0 && container != nil:
		name = fmt.Sprintf("%s%s", container.Name(), lower_case_to_camel_case(name))
	default:
		name = fmt.Sprintf("%s", lower_case_to_camel_case(name))
	}
	fullnm += name
	name = g_config.rename(fullnm, name)
	printf("%s(", name)
	for i, arg := range fb.args {
		printf("%s0 %s", arg.arg_info.Name(), go_type(arg.type_info, type_none))
		if i != len(fb.args)-1 {
			printf(", ")
		}
	}
	printf(")")
	switch len(fb.rets) {
	case 0:
		// do nothing if there are not return values
	case 1:
		if flags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
			// override return types for constructors, We can't
			// return generic widget here as C does. Go's type
			// system is stronger.
			printf(" %s",
				go_type_for_interface(container, type_pointer|type_return))
			break
		}
		if fb.rets[0].index == -2 {
			printf(" error")
		} else {
			printf(" %s", go_type(fb.rets[0].type_info, type_return))
		}
	default:
		printf(" (")
		for i, ret := range fb.rets {
			if ret.index == -2 {
				// special error type in Go to represent GError
				printf("error")
				continue
			}
			printf(go_type(ret.type_info, type_return))
			if i != len(fb.rets)-1 {
				printf(", ")
			}
		}
		printf(")")
	}
	printf(" {\n")

	// --- body stage 1 (Go to C conversions)

	// var declarations
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("\tvar this1 %s\n", cgo_type_for_interface(container, type_pointer))
	}
	for _, arg := range fb.args {
		printf("\tvar %s1 %s\n", arg.arg_info.Name(),
			cgo_type(arg.type_info, type_none))
		if al := arg.type_info.ArrayLength(); al != -1 {
			arg := fb.orig_args[al]
			printf("\tvar %s1 %s\n", arg.Name(),
				cgo_type(arg.Type(), type_none))
		}
	}

	for _, ret := range fb.rets {
		if ret.index == -1 {
			continue
		}

		if ret.index == -2 {
			printf("\tvar err1 *C.GError\n")
			continue
		}

		if ret.arg_info.Direction() == gi.DIRECTION_INOUT {
			continue
		}

		printf("\tvar %s1 %s\n", ret.arg_info.Name(),
			cgo_type(ret.type_info, type_none))
		if al := ret.type_info.ArrayLength(); al != -1 {
			arg := fb.orig_args[al]
			printf("\tvar %s1 %s\n", arg.Name(),
				cgo_type(arg.Type(), type_none))
		}
	}

	// conversions
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		conv := go_to_cgo_for_interface(container, "this0", "this1", conv_pointer)
		printf("%s", print_lines_with_indent(conv))
	}
	for _, arg := range fb.args {
		nm := arg.arg_info.Name()
		conv := go_to_cgo(arg.type_info, nm+"0", nm+"1", conv_none)
		printf("%s", print_lines_with_indent(conv))

		// register callback in the global map
		if arg.type_info.Tag() == gi.TYPE_TAG_INTERFACE {
			bi := arg.type_info.Interface()
			if bi.Type() == gi.INFO_TYPE_CALLBACK {
				if arg.arg_info.Scope() != gi.SCOPE_TYPE_CALL {
					printf("\t%sHolder.Grab(%s1)\n", g_config.sys.gns, nm)
				}
			}
		}

		// array length
		if len := arg.type_info.ArrayLength(); len != -1 {
			lenarg := fb.orig_args[len]
			conv = go_to_cgo(lenarg.Type(),
				"len("+nm+"0)", lenarg.Name()+"1", conv_none)
			printf("\t%s\n", conv)
		}
	}

	// --- body stage 2 (the function call)
	printf("\t")
	if fb.has_return_value() {
		printf("ret1 := ")
	}

	userdata, destroy, scope := fb.has_closure_argument()
	printf("C.")
	if scope != gi.SCOPE_TYPE_INVALID {
		printf("_")
	}
	printf("%s(", fi.Symbol())
	if flags&gi.FUNCTION_IS_METHOD != 0 {
		printf("this1")
		if len(fb.orig_args) > 0 {
			printf(", ")
		}
	}
	for i, ri, n := 0, 0, len(fb.orig_args); i < n; i++ {
		if i == userdata || i == destroy {
			continue
		}

		oarg := fb.orig_args[i]

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
	for _, ret := range fb.rets {
		switch ret.index {
		case -1:
			if flags&gi.FUNCTION_IS_CONSTRUCTOR != 0 {
				printf("\tvar ret2 %s\n",
					go_type_for_interface(container, type_pointer|type_return))
			} else {
				printf("\tvar ret2 %s\n", go_type(ret.type_info, type_return))
			}
		case -2:
			printf("\tvar err2 error\n")
		default:
			printf("\tvar %s2 %s\n", ret.arg_info.Name(),
				go_type(ret.type_info, type_return))
		}
	}

	// conversions
	for _, ret := range fb.rets {
		if ret.index == -2 {
			printf("\tif err1 != nil {\n")
			printf("\t\terr2 = errors.New(C.GoString(((*_GError)(unsafe.Pointer(err1))).message))\n")
			printf("\t\tC.g_error_free(err1)\n")
			printf("\t}\n")
			continue
		}

		var nm string
		if ret.index == -1 {
			nm = "ret"
		} else {
			nm = ret.arg_info.Name()
		}

		// array length
		if len := ret.type_info.ArrayLength(); len != -1 {
			lenarg := fb.orig_args[len]
			printf("\t%s2 = make(%s, %s)\n",
				nm, go_type(ret.type_info, type_return),
				lenarg.Name()+"1")
		}

		var ownership gi.Transfer
		if ret.index == -1 {
			ownership = fi.CallerOwns()
		} else {
			ownership = ret.arg_info.OwnershipTransfer()
			if ret.arg_info.Direction() == gi.DIRECTION_INOUT {
				// not sure if it's true in all cases, but so far
				ownership = gi.TRANSFER_NOTHING
			}
		}
		var conv string
		if flags&gi.FUNCTION_IS_CONSTRUCTOR != 0 && ret.index == -1 {
			conv = cgo_to_go_for_interface(container, "ret1", "ret2",
				conv_pointer|ownership_to_conv_flags(ownership))
		} else {
			conv = cgo_to_go(ret.type_info, nm+"1", nm+"2",
				ownership_to_conv_flags(ownership))
		}
		printf("%s", print_lines_with_indent(conv))
	}

	// --- body stage 4 (return)
	if len(fb.rets) == 0 {
		printf("}\n")
		return
	}

	printf("\treturn ")
	for i, ret := range fb.rets {
		var nm string
		switch ret.index {
		case -1:
			nm = "ret2"
		case -2:
			nm = "err2"
		default:
			nm = ret.arg_info.Name() + "2"
		}
		if i != 0 {
			printf(", ")
		}
		printf(nm)
	}

	printf("\n}\n")
}

func process_interface_info(ii *gi.InterfaceInfo) {
	name := ii.Name()
	cprefix := gi.DefaultRepository().CPrefix(ii.Namespace())
	cgotype := cgo_type_for_interface(gi.ToBaseInfo(ii), type_pointer)

	printf("%s\n", execute_template(interface_template, map[string]string{
		"name":     name,
		"cprefix":  cprefix,
		"cgotype":  cgotype,
		"typeinit": ii.TypeInit(),
		"gobjectns": g_config.sys.gns,
	}))

	for i, n := 0, ii.NumMethod(); i < n; i++ {
		meth := ii.Method(i)
		if g_config.is_method_blacklisted(name, meth.Name()) {
			printf("// blacklisted: %s.%s (method)\n", name, meth.Name())
			continue
		}
		process_function_info(meth, gi.ToBaseInfo(ii))
	}
}

func process_base_info(bi *gi.BaseInfo) {
	switch bi.Type() {
	case gi.INFO_TYPE_UNION:
		if g_config.is_blacklisted("unions", bi.Name()) {
			goto blacklisted
		}
		process_union_info(gi.ToUnionInfo(bi))
	case gi.INFO_TYPE_STRUCT:
		if g_config.is_blacklisted("structs", bi.Name()) {
			goto blacklisted
		}
		process_struct_info(gi.ToStructInfo(bi))
	case gi.INFO_TYPE_ENUM, gi.INFO_TYPE_FLAGS:
		if g_config.is_blacklisted("enums", bi.Name()) {
			goto blacklisted
		}
		process_enum_info(gi.ToEnumInfo(bi))
	case gi.INFO_TYPE_CONSTANT:
		if g_config.is_blacklisted("constants", bi.Name()) {
			goto blacklisted
		}
		process_constant_info(gi.ToConstantInfo(bi))
	case gi.INFO_TYPE_CALLBACK:
		if g_config.is_blacklisted("callbacks", bi.Name()) {
			goto blacklisted
		}
		process_callback_info(gi.ToCallableInfo(bi))
	case gi.INFO_TYPE_FUNCTION:
		if g_config.is_blacklisted("functions", bi.Name()) {
			goto blacklisted
		}
		process_function_info(gi.ToFunctionInfo(bi), nil)
	case gi.INFO_TYPE_INTERFACE:
		if g_config.is_blacklisted("interfaces", bi.Name()) {
			goto blacklisted
		}
		process_interface_info(gi.ToInterfaceInfo(bi))
	case gi.INFO_TYPE_OBJECT:
		if g_config.is_blacklisted("objects", bi.Name()) {
			goto blacklisted
		}
		process_object_info(gi.ToObjectInfo(bi))
	default:
		printf("// TODO: %s (%s)\n", bi.Name(), bi.Type())
	}
	return

blacklisted:
	printf("// blacklisted: %s (%s)\n", bi.Name(), bi.Type())
}
