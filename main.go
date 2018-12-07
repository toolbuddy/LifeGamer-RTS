package main

import (
	"flag"

	"comm"
	"config"
	"game"
	"util"
)

func main() {

	genJson := flag.Bool("genjson", false, "Generate protocal json")
	configPath := flag.String("config", "src/config/default.json", "Path to configuration file")
	wdbPath := flag.String("wdb", "", "Path to world database")
	pdbPath := flag.String("pdb", "", "Path to player database")
	hostname := flag.String("host", "", "Name of login host")

	flag.Parse()

	if *genJson {
		comm.MsgType2Json()
		return
	}

	config.Initialize(*configPath,
		map[string]interface{}{
			config.IDHostname: *hostname,
			config.IDWdbPath:  *wdbPath,
			config.IDPdbPath:  *pdbPath,
        })

	server, _ := comm.NewWsServer()
	server.Start(9999)

	engine, _ := game.NewGameEngine()
	engine.LoadTerrain(util.Point{-25, -25}, util.Point{24, 24}, "map.json")
	engine.Start()

	//reader := bufio.NewReader(os.Stdin)
	//for {
	//reader.ReadString('\n')
	//}
	select {}
}
