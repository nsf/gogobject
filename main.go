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

var GConfigPath = flag.String("config", "", "specify global config file")

// per namespace config file
var Config struct {
	Namespace       string              `json:"namespace"`
	Version         string              `json:"version"`
	Blacklist       map[string][]string `json:"blacklist"`
	Whitelist       map[string][]string `json:"whitelist"`
	MethodBlacklist map[string][]string `json:"method-blacklist"`
	MethodWhitelist map[string][]string `json:"method-whitelist"`
	Renames         map[string]string   `json:"renames"`

	// variables that are calculated during the app execution
	Sys struct {
		Out             *bufio.Writer
		Outdir          string
		Package         string
		Blacklist       map[string]map[string]bool
		Whitelist       map[string]map[string]bool
		MethodBlacklist map[string]map[string]bool
		MethodWhitelist map[string]map[string]bool

		// "gobject." if the current namespace is not GObject, "" otherwise
		GNS string
	} `json:"-"`
}

// global config file
var GConfig struct {
	DisguisedTypes []string `json:"disguised-types"`

	Sys struct {
		DisguisedTypes map[string]bool
	} `json:"-"`
}

func Rename(path, oldname string) string {
	if newname, ok := Config.Renames[path]; ok {
		return newname
	}
	return oldname
}

func IsBlacklisted(section, entry string) bool {
	// check if the entry is in the blacklist
	if sectionMap, ok := Config.Sys.Blacklist[section]; ok {
		if _, ok := sectionMap[entry]; ok {
			return true
		}
	}

	// check if the entry is missing from the whitelist
	if sectionMap, ok := Config.Sys.Whitelist[section]; ok {
		if _, ok := sectionMap[entry]; !ok {
			return true
		}
	}

	return false
}

func IsMethodBlacklisted(class, method string) bool {
	if classMap, ok := Config.Sys.MethodBlacklist[class]; ok {
		if _, ok := classMap[method]; ok {
			return true
		}
	}

	if classMap, ok := Config.Sys.MethodWhitelist[class]; ok {
		if _, ok := classMap[method]; !ok {
			return true
		}
	}

	return false
}

func ParseJSONWithComments(filename string, data interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	d := json.NewDecoder(NewCommentSkipper(f))
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

	// parse global config
	if *GConfigPath != "" {
		err := ParseJSONWithComments(*GConfigPath, &GConfig)
		if err != nil {
			panic(err)
		}

		GConfig.Sys.DisguisedTypes = ListToMap(GConfig.DisguisedTypes)
	}

	// parse config
	configPath := filepath.Join(flag.Arg(0), "config.json")
	err := ParseJSONWithComments(configPath, &Config)
	if err != nil {
		panic(err)
	}

	repo := gi.DefaultRepository()

	// load namespace
	_, err = repo.Require(Config.Namespace, Config.Version, 0)
	if err != nil {
		panic(err)
	}

	// setup some of the Sys vars
	Config.Sys.Package = strings.ToLower(Config.Namespace)
	Config.Sys.Outdir = filepath.Clean(flag.Arg(0))
	Config.Sys.Whitelist = MapListToMapMap(Config.Whitelist)
	Config.Sys.Blacklist = MapListToMapMap(Config.Blacklist)
	Config.Sys.MethodWhitelist = MapListToMapMap(Config.MethodWhitelist)
	Config.Sys.MethodBlacklist = MapListToMapMap(Config.MethodBlacklist)

	if Config.Namespace != "GObject" {
		Config.Sys.GNS = "gobject."
	}

	// prepare main output
	filename := filepath.Join(Config.Sys.Outdir,
		strings.ToLower(Config.Namespace)+".go")
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	Config.Sys.Out = bufio.NewWriter(file)

	tpl, err := ioutil.ReadFile(filename + ".in")
	if err != nil {
		panic(err)
	}

	ProcessTemplate(string(tpl))

	Config.Sys.Out.Flush()
}
