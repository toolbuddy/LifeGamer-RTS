package main

import (
    "comm"
    "bufio"
    "os"
    "encoding/json"
    "game"
)

func main() {
    mbus, _ := comm.NewMBusNode("main")
    reader := bufio.NewReader(os.Stdin)

    server, _ := comm.NewWsServer()
    server.Start(9999)

    engine, _ := game.NewGameEngine()
    engine.Start()

    for {
        msg, _ := reader.ReadString('\n')
        b, _ := json.Marshal(comm.Payload { comm.PlayerDataResponse, "HMKRL", msg })
        mbus.Write("ws", b)
    }
}
