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
	configPath := flag.String("config", "src/config/default.json", "Path to configuration file")

	hostname := flag.String("hostname", "", "Login hostname")
	db_dir := flag.String("db_dir", "", "Directory of game database")
	log_dir := flag.String("log_dir", "", "Directory of log file")

	verbose := flag.Bool("verbose", false, "Whether to log filename and line number out")

	flag.Parse()

	if *genJson {
		comm.MsgType2Json()
		return
	}

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	config.Initialize(*configPath,
		map[string]interface{}{
			config.IDHostname: *hostname,
			config.IDDBDir:    *db_dir,
			config.IDLogDir:   *log_dir,
		})

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
