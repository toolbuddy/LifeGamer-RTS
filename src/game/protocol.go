package game

import (
    "comm"
    "util"
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
    Restart     SAction = "Restart"
)

type PlayerDataPayload struct {
    comm.Payload
    player.Player
}

type MapDataPayload struct {
    comm.Payload
    Chunks []world.Chunk
}

type MinimapDataPayload struct {
    comm.Payload
    Size util.Size
    Terrain [][]world.TerrainType
    Owner   [][]string
}

type BuildingPayload struct {
    comm.Payload
    Action SAction
    Structure world.Structure
}
