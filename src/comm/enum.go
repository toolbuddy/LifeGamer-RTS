package comm

import (
)

type RequestType uint

const (
    GetPlayerData RequestType = iota
    GetMapData
)

type PushType uint

const (
    Push = iota
)
