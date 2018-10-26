package comm

import (
    "game/util"
)

type MsgType uint // request from browser

const (
    PlayerData MsgType = iota
    MapData
)

type BasePayload struct {
    Msg_type MsgType
    Username string
}

type PlayerDataPayload struct {
    BasePayload

    Human int
    Money int
    Power int
}

type MapDataPayload struct {
    BasePayload

    Chunk util.Point
    // TODO: add map data ...
}

type BuildPayload struct {
    BasePayload

    Chunk util.Point
    Pos util.Point
    // TODO: add structure type ...
}
