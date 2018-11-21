package main

import (
    "comm"
    //"bufio"
    //"os"
    "game"
)

func main() {
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
