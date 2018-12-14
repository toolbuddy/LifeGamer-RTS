package player

import (
	"time"
	"util"
)

type Player struct {
	Population     int64
	PopulationCap  int64
	PopulationRate int64

	Money     int64
	MoneyRate int64

	Power    int64
	PowerMax int64

	Home        util.Point // spawn point
	Initialized bool
	UpdateTime  int64 // Unix time
}

// Update player's current data based on current time & previous update time
// TODO: Burst Link
func (player *Player) Update() {
	current := time.Now().Unix()
	player.Money += player.MoneyRate * (current - player.UpdateTime)
	player.Population = func() int64 {
		result := player.Population + player.PopulationRate*(current-player.UpdateTime)
		if result > player.PopulationCap {
			result = player.PopulationCap
		}

		return result
	}()
	player.UpdateTime = current
	return
}
