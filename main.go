package main

import (
    "comm"
    "bufio"
    "fmt"
    "os"
)

func main() {
    comm.WsServerStart(9999)
    mbus := comm.NewClient("main")
    reader := bufio.NewReader(os.Stdin)

    for {
        fmt.Print("Enter text: ")
        text, _ := reader.ReadString('\n')
        //fmt.Println(text)
        mbus.Put("ws", text)
    }
}
