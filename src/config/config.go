package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

const (
	IDHostname = "hostname"
	IDWdbPath  = "wdb_path"
	IDPdbPath  = "pdb_path"
)

var (
	Hostname string
	WdbPath  string
	PdbPath  string
)

// Initialize : Load default config and override with data
func Initialize(path string, data map[string]interface{}) {

	// Read default configuration
	v, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("[WARNING] Failed to read config file")
	}

	var defaultConfig map[string]interface{}
	err = json.Unmarshal([]byte(v), &defaultConfig)
	if err != nil {
		log.Println("[WARNING] Failed to unmarshal config file")
	}

	// Apply default config
	apply(defaultConfig)

	// Override config (if specified)
	apply(data)

	log.Printf("Using config from %v:"+strings.Repeat("\n\t%v : %v", 3)+"\n",
		path,
		IDHostname, Hostname,
		IDWdbPath, WdbPath,
		IDPdbPath, PdbPath)

	// Verify config
	if containsEmpty(Hostname, WdbPath, PdbPath) {
		log.Fatalln("[ERROR] Invalid Configuration")
	}

}

func apply(data map[string]interface{}) {
	for k := range data {
		if s := fmt.Sprintf("%v", data[k]); s != "" {
			switch k {
			case IDHostname:
				Hostname = s
			case IDWdbPath:
				WdbPath = s
			case IDPdbPath:
				PdbPath = s
			}
		}
	}
}

func containsEmpty(ss ...string) bool {
	for _, s := range ss {
		if s == "" {
			return true
		}
	}
	return false
}
