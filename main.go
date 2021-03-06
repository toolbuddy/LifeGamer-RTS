package main

import (
	"comm"
	"config"
	"flag"
	"game"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"time"
	"util"
)

func main() {
	genJson := flag.Bool("genjson", false, "Generate protocal json")

	// configure options
	configPath := flag.String("config", "src/config/default.json", "Path to configuration file")

	verbose := flag.Bool("verbose", false, "Whether to log filename and line number out")

	debuguser := flag.String("user", "", "Skip login and use this username")

	flag.Parse()

	if *genJson {
		comm.MsgType2Json()
		return
	}

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if *debuguser != "" {
		log.Printf("Using username %v for debug", *debuguser)
		os.Setenv("RTSUSER", *debuguser)
	}

	config.Initialize(*configPath)

	// Create log directory
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		log.Fatalln("[ERROR] Unable to create log directory")
		return
	}

	log_path := path.Join(config.LogDir, "gamelog_"+strconv.Itoa(int(time.Now().Unix()))+".log")
	fileWriter, err := os.OpenFile(log_path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln("[ERROR] Unable to create log file")
		return
	}

	log.SetOutput(io.MultiWriter(os.Stdout, fileWriter))

	engine, _ := game.NewGameEngine()
	engine.LoadTerrain(util.Point{-25, -25}, util.Point{24, 24}, "map_river.json")
	engine.Start()

	server, _ := comm.NewWsServer()
	server.Start(9999)

	select {}
}
