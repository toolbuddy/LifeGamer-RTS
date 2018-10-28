package main

import (
    "comm"
    "bufio"
    "fmt"
    "os"
)

func main() {
    mbus, _ := comm.NewMBusNode("main")
    reader := bufio.NewReader(os.Stdin)
    server, _ := comm.NewWsServer()

    server.Start(9999)

    for {
        fmt.Print("Enter text: ")
        text, _ := reader.ReadString('\n')
        mbus.Write("ws", comm.BasePayload { comm.PlayerData, "HMKRL", text })
    }
}
