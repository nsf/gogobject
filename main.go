package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"gobject/gi"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var g_commonconfig_path = flag.String("config", "", "specify global config file")
var g_output_dir = flag.String("o", "", "override output directory")

// per namespace config file
var g_config config
var g_commonconfig commonconfig

type config struct {
	Namespace       string              `json:"namespace"`
	Version         string              `json:"version"`
	Blacklist       map[string][]string `json:"blacklist"`
	Whitelist       map[string][]string `json:"whitelist"`
	MethodBlacklist map[string][]string `json:"method-blacklist"`
	MethodWhitelist map[string][]string `json:"method-whitelist"`
	Renames         map[string]string   `json:"renames"`

	// variables that are calculated during the app execution
	sys struct {
		out             *bufio.Writer
		pkg             string
		blacklist       map[string]map[string]bool
		whitelist       map[string]map[string]bool
		method_blacklist map[string]map[string]bool
		method_whitelist map[string]map[string]bool

		// "gobject." if the current namespace is not GObject, "" otherwise
		gns string
	} `json:"-"`
}

func (this *config) load(path string) {
	err := parse_json_with_comments(path, this)
	if err != nil {
		panic(err)
	}

	this.sys.pkg = strings.ToLower(this.Namespace)
	this.sys.whitelist = map_list_to_map_map(this.Whitelist)
	this.sys.blacklist = map_list_to_map_map(this.Blacklist)
	this.sys.method_whitelist = map_list_to_map_map(this.MethodWhitelist)
	this.sys.method_blacklist = map_list_to_map_map(this.MethodBlacklist)

	if this.Namespace != "GObject" {
		this.sys.gns = "gobject."
	}

}

func (this *config) rename(path, oldname string) string {
	if newname, ok := this.Renames[path]; ok {
		return newname
	}
	return oldname
}

func (this *config) is_blacklisted(section, entry string) bool {
	// check if the entry is in the blacklist
	if section_map, ok := this.sys.blacklist[section]; ok {
		if _, ok := section_map[entry]; ok {
			return true
		}
	}

	// check if the entry is missing from the whitelist
	if section_map, ok := this.sys.whitelist[section]; ok {
		if _, ok := section_map[entry]; !ok {
			return true
		}
	}

	return false
}

func (this *config) is_method_blacklisted(class, method string) bool {
	// don't want to see these
	if method == "ref" || method == "unref" {
		return true
	}

	if class_map, ok := this.sys.method_blacklist[class]; ok {
		if _, ok := class_map[method]; ok {
			return true
		}
	}

	if class_map, ok := this.sys.method_whitelist[class]; ok {
		if _, ok := class_map[method]; !ok {
			return true
		}
	}

	return false
}

// global config file
type commonconfig struct {
	DisguisedTypes []string `json:"disguised-types"`
	WordSubst map[string]string `json:"word-subst"`

	sys struct {
		disguised_types map[string]bool
	} `json:"-"`
}

func (this *commonconfig) load(path string) {
	err := parse_json_with_comments(path, this)
	if err != nil {
		panic(err)
	}
	this.sys.disguised_types = list_to_map(this.DisguisedTypes)
}

func parse_json_with_comments(filename string, data interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	d := json.NewDecoder(new_comment_skipper(f))
	err = d.Decode(data)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <dir>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		return
	}

	// parse common config file
	if *g_commonconfig_path != "" {
		g_commonconfig.load(*g_commonconfig_path)
	}

	// figure in/out paths
	in_dir, in_file := filepath.Split(flag.Arg(0))
	in_path := flag.Arg(0)

	out_dir := in_dir
	if *g_output_dir != "" {
		out_dir = *g_output_dir
	}
	out_file := in_file[:len(in_file)-3]
	out_path := filepath.Join(out_dir, out_file)

	// parse local config
	g_config.load(filepath.Join(in_dir, "config.json"))


	repo := gi.DefaultRepository()

	// load namespace
	_, err := repo.Require(g_config.Namespace, g_config.Version, 0)
	if err != nil {
		panic(err)
	}

	// prepare main output
	file, err := os.Create(out_path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	g_config.sys.out = bufio.NewWriter(file)

	tpl, err := ioutil.ReadFile(in_path)
	if err != nil {
		panic(err)
	}

	process_template(string(tpl))

	g_config.sys.out.Flush()
}
