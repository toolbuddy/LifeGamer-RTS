package player

import (
	"time"
	"util"
)

type Player struct {
	Population    int64
	PopulationCap int64

	Money     int64
	MoneyRate int64

	Power    int64
	PowerMax int64

	Home      util.Point // spawn point
	Territory []util.Point

	Initialized bool
	UpdateTime  int64 // Unix time
}

func NewPlayer() *Player {
	return &Player{Territory: []util.Point{}}
}

// Update player's current data based on current time & previous update time
// TODO: Burst Link
func (player *Player) Update() {
	current := time.Now().Unix()
	player.Money += player.MoneyRate * (current - player.UpdateTime)
	player.UpdateTime = current
	return
}

// Same as update, but don't modify original player object
func (player Player) GetStatus() Player {
	current := time.Now().Unix()
	player.Money += player.MoneyRate * (current - player.UpdateTime)

	return player
}
