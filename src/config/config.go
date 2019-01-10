package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"
)

const (
	idHostname = "hostname"
	idDBDir    = "db_dir"
	idLogDir   = "log_dir"
)

var (
	Hostname string
	DBDir    string
	LogDir   string
)

// Initialize : Load default config and override with data
func Initialize(path string) {

	// Read default configuration
	v, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("[WARNING] Failed to read config file")
	}

	var configData map[string]interface{}
	err = json.Unmarshal([]byte(v), &configData)
	if err != nil {
		log.Println("[WARNING] Failed to unmarshal config file")
	}

	apply(configData)

	log.Printf("[INFO] Using config from %v:"+strings.Repeat("\n\t%v : %v", 3)+"\n",
		path,
		idHostname, Hostname,
		idDBDir, DBDir,
		idLogDir, LogDir)

	// Verify config
	if msglist := verify(); len(msglist) > 0 {
		for _, msg := range msglist {
			log.Println("[INFO] " + msg)
		}
		log.Fatalln("[ERROR] Aborting due to invalid config.")
	}

}

func apply(data map[string]interface{}) {
	for k, v := range data {
		switch v.(type) {
		case string:
			s := v.(string)
			switch k {
			case idHostname:
				Hostname = s
			case idDBDir:
				DBDir = s
			case idLogDir:
				LogDir = s
			}
		}
	}
}

// Returns slice of strings containing error message
func verify() (msglist []string) {

	// Specify the limitations of config.
	// e.g. "numbers can't be 0", "address not responding", etc.
	cannotBeBlank := " cannot be blank."

	if Hostname == "" {
		msglist = append(msglist, "\""+idHostname+"\""+cannotBeBlank)
	}

	if DBDir == "" {
		msglist = append(msglist, "\""+idDBDir+"\""+cannotBeBlank)
	}

	if LogDir == "" {
		msglist = append(msglist, "\""+idLogDir+"\""+cannotBeBlank)
	}

	return
}
