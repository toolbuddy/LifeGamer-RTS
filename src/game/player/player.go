package player

import (
    "util"
    "time"
)

type Player struct {
    Human int64
    Money int64
    Power int64
    Home util.Point     // spawn point
    Initialized bool
    UpdateTime int64    // Unix time
}

// Update player's current data based on current time & previous update time
// TODO: Use real change rate
func (player *Player) Update() {
    current := time.Now().Unix()
    player.Money += 100 * (current - player.UpdateTime)
    player.UpdateTime = current
    return
}
