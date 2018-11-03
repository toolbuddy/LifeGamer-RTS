package main

import (
    "comm"
    "bufio"
    "fmt"
    "os"
    "encoding/json"
)

func main() {
    mbus, _ := comm.NewMBusNode("main")
    reader := bufio.NewReader(os.Stdin)
    server, _ := comm.NewWsServer()

    server.Start(9999)

    for {
        fmt.Print("Enter text: ")
        msg, _ := reader.ReadString('\n')
        b, _ := json.Marshal(comm.Payload { comm.PlayerDataResponse, "HMKRL", msg })
        mbus.Write("wss", b)
    }
}
