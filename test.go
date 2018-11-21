package main

import (
    "game/world"
    "io/ioutil"
    "log"
    "fmt"
)

func main() {
    dat, err := ioutil.ReadFile("/tmp/structures.json")
    sdef, err := world.LoadDefinition(dat)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(sdef)
}
