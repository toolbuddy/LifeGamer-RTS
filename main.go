package main

import (
    "comm"
    //"bufio"
    //"os"
    "flag"
    "game"
    "util"
)

func main() {
    genJson := flag.Bool("genjson", false, "Generate protocal json")
    flag.Parse()

    if *genJson {
        comm.MsgType2Json()
        return
    }

    server, _ := comm.NewWsServer()
    server.Start(9999)

    engine, _ := game.NewGameEngine()
    engine.LoadTerrain(util.Point{-10, -10}, util.Point{9, 9}, "map.json")
    engine.Start()

    //reader := bufio.NewReader(os.Stdin)
    //for {
        //reader.ReadString('\n')
    //}
    select {}
}
