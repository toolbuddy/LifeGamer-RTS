package player

import (
	"time"
	"util"
)

type Player struct {
	Human     int64
	HumanMax  int64
	HumanRate int64

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
	player.Human = func() int64 {
		result := player.Human + player.HumanRate*(current-player.UpdateTime)
		if result > player.HumanMax {
			result = player.HumanMax
		}

		return result
	}()
	player.UpdateTime = current
	return
}
