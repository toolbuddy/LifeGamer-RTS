package main

import (
    "comm"
    "bufio"
    "fmt"
    "os"
)

func main() {
    mbus := comm.NewMBusNode("main")
    comm.WsServerStart(9999)
    reader := bufio.NewReader(os.Stdin)

    for {
        fmt.Print("Enter text: ")
        text, _ := reader.ReadString('\n')
        //fmt.Println(text)
        mbus.Put("ws", text)
    }
}
