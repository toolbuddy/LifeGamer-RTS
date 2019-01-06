package game

import (
	"comm"
	"encoding/json"
	"fmt"
	"game/world"
	"log"
	"time"
	"util"
)

// This file is used to notify when data updates

type Notifier struct {
	GameDB
	CommonData

	mbus *comm.MBusNode
}

func NewNotifier(gameDB GameDB, common_data CommonData, mbus *comm.MBusNode) (notifier *Notifier) {
	notifier = &Notifier{
		gameDB,
		common_data,
		mbus,
	}

	return
}

func (notifier Notifier) start() {
	// start ticker
	go func() {
		// Align update tick to second
		<-time.After(time.Until(time.Now().Add(time.Second).Truncate(time.Second)))
		for range time.NewTicker(time.Second).C {
			notifier.onlineLock.RLock()
			for _, user_ch := range notifier.online_players {
				user_ch <- ""
			}
			notifier.onlineLock.RUnlock()
		}
	}()

	// player DB update checking
	go func() {
		for username := range notifier.playerDB.Updated {
			notifier.onlineLock.RLock()
			if user_ch, ok := notifier.online_players[username]; ok {
				user_ch <- "update"
			}
			notifier.onlineLock.RUnlock()
		}
	}()

	// world DB update checking
	go func() {
		for spos := range notifier.worldDB.Updated {
			pos := util.Point{}
			fmt.Sscanf(spos, "%d,%d", &pos.X, &pos.Y)
			notifier.mapDataUpdate(pos)
		}
	}()

	// chunk owner change checking
	go func() {
		for key := range notifier.owner_changed {
			chk, err := notifier.worldDB.Get(key)
			if err != nil {
				log.Println("[WARNING]", err)
				continue
			}

			var x, y int
			fmt.Sscanf(key, "%d,%d", &x, &y)

			notifier.minimapLock.Lock()
			notifier.minimap.Owner[x+25][y+25] = chk.Owner
			payload := MinimapDataPayload{comm.Payload{Msg_type: comm.MinimapDataResponse}, *notifier.minimap}
			notifier.minimapLock.Unlock()

			b, err := json.Marshal(payload)
			if err != nil {
				log.Println("[WARNING]", err)
				continue
			}

			msg := comm.MessageWrapper{SendTo: comm.Broadcast, Data: b}
			notifier.mbus.Write("ws", msg)
		}
	}()
}

func (notifier Notifier) mapDataUpdate(position util.Point) {
	// read which clients are watching this chunk
	notifier.chunkLock.RLock()
	infos, ok := notifier.chunk2Clients[position]
	if !ok {
		notifier.chunkLock.RUnlock()
		return
	}
	notifier.chunkLock.RUnlock()

	// update the map of these clients
	for _, info := range infos {
		chunks := []world.Chunk{}

		// read chunks that the client is watching
		notifier.clientLock.RLock()
		poss, ok := notifier.client2Chunks[info]
		if !ok {
			notifier.clientLock.RUnlock()
			continue
		}
		notifier.clientLock.RUnlock()

		// load chunks data from DB
		for _, pos := range poss {
			chunk, err := notifier.worldDB.Get(pos.String())
			if err != nil {
				log.Println("[WARNING]", err)
				continue
			}

			if pos == position {
				// Population not enough, halt this chunk and cancel current message
				if chunk.Population < chunk.PopulationNeed() {
					defer HaltChunk(notifier.GameDB, info.username, pos.String())
				}

				// Minimap data update
				notifier.minimapLock.RLock()
				if chunk.Owner != notifier.minimap.Owner[pos.X+25][pos.Y+25] {
					notifier.owner_changed <- pos.String()
				}
				notifier.minimapLock.RUnlock()
			}

			chunks = append(chunks, chunk)
		}

		// send data to client
		payload := comm.Payload{Msg_type: comm.MapDataResponse}
		map_data := MapDataPayload{payload, chunks}

		b, err := json.Marshal(map_data)
		if err != nil {
			log.Println("[WARNING]", err)
			continue
		}

		msg := comm.MessageWrapper{info.cid, info.username, comm.SendToClient, b}

		notifier.mbus.Write("ws", msg)
	}
}

func playerDataUpdate(client_info ClientInfo, user_ch <-chan string, mbus *comm.MBusNode, db GameDB) {
	username := client_info.username

	player_data, err := db.playerDB.Get(username)
	if err != nil {
		log.Println("[WARNING]", err)
		return
	}

	// Update player's data
	player_data.Update()

	for m := range user_ch {
		if m == "update" {
			player_data, err = db.playerDB.Get(username)
			if err != nil {
				log.Println("[WARNING]", err)
				continue
			}
		}

		current_status := player_data.GetStatus()

		if current_status.Money < 0 || current_status.Power > current_status.PowerMax {
			// No enough money or power for player
			HaltPlayer(db, username)
		} else {
			b, err := json.Marshal(PlayerDataPayload{comm.Payload{comm.PlayerDataResponse}, current_status})
			if err != nil {
				log.Println("[WARNING]", err)
				continue
			}

			msg := comm.MessageWrapper{client_info.cid, username, comm.SendToUser, b}

			mbus.Write("ws", msg)
		}
	}

	log.Printf("[INFO] the data update of player \"%s\" has stopped", username)
}
