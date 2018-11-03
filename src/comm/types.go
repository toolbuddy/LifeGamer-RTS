package comm

import (
)

type MsgType uint // message type

const (
    LoginRequest MsgType = iota
    PlayerDataRequest
    MapDataRequest

    LoginResponse
    PlayerDataResponse
    MapDataResponse
)

type Payload struct {
    Msg_type MsgType
    Username string
    Message  string
}

type PlayerDataPayload struct {
    Payload

    Human int
    Money int
    Power int
}
