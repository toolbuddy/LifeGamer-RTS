package main

import (
    "comm"
    "game"
)

func main() {
    server, _ := comm.NewWsServer()
    server.Start(9999)

    engine, _ := game.NewGameEngine()
    engine.Start()

    for {
    }
}
