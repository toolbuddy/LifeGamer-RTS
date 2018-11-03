package comm

import (
    "util"
)

type MsgType uint // message type

const (
    LoginRequest MsgType = iota
    PlayerDataRequest
    MapDataRequest
    HomePointRequest

    LoginResponse
    PlayerDataResponse
    MapDataResponse
    HomePointResponse
)

type Payload struct {
    Msg_type MsgType
    Username string
    Message  string
}

type PlayerDataPayload struct {
    Payload

    Home util.Point // spawn point
    Human int
    Money int
    Power int
}
