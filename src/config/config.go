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
	IDDBDir    = "db_dir"
	IDLogDir   = "log_dir"
)

var (
	Hostname string
	DBDir    string
	LogDir   string
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

	log.Printf("[INFO] Using config from %v:"+strings.Repeat("\n\t%v : %v", 3)+"\n",
		path,
		IDHostname, Hostname,
		IDDBDir, DBDir,
		IDLogDir, LogDir)

	// Verify config
	if containsEmpty(Hostname, DBDir, LogDir) {
		log.Fatalln("[ERROR] Invalid Configuration")
	}

}

func apply(data map[string]interface{}) {
	for k, v := range data {
		if s := fmt.Sprintf("%v", v); s != "" {
			switch k {
			case IDHostname:
				Hostname = s
			case IDDBDir:
				DBDir = s
			case IDLogDir:
				LogDir = s
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
