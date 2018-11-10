package game

import (
    "comm"
    "game/player"
)

type PlayerDataPayload struct {
    comm.Payload
    player.Player
}
