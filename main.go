package main

import (
    "comm"
    //"bufio"
    //"os"
    "flag"
    "game"
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
    engine.Start()

    //reader := bufio.NewReader(os.Stdin)
    //for {
        //reader.ReadString('\n')
    //}
    select {}
}
