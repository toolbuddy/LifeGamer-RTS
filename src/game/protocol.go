package game

import (
    "comm"
    "game/player"
    "game/world"
)

type PlayerDataPayload struct {
    comm.Payload
    player.Player
}

type MapDataPayload struct {
    comm.Payload
    Chunks []world.Chunk
}
