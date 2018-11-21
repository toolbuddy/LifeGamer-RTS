package game

import (
    "comm"
    "game/player"
    "game/world"
)

// Structure action
type SAction string

const (
    Build       SAction = "Build"
    Upgrade     SAction = "Upgrade"
    Destruct    SAction = "Destruct"
    Repair      SAction = "Repair"
)

type PlayerDataPayload struct {
    comm.Payload
    player.Player
}

type MapDataPayload struct {
    comm.Payload
    Chunks []world.Chunk
}

type BuildingPayload struct {
    comm.Payload
    Action SAction
    Structure world.Structure
}
