package world

import (
	"util"
)

// Structure status
type SStatus string

const (
	Running     SStatus = "Running"     // Running normally
	Building    SStatus = "Building"    // building or upgrading
	Destructing SStatus = "Destructing" // Being destruting by player
	Destroyed   SStatus = "Destroyed"   // Destroyed after war
	Halted      SStatus = "Halted"      // Halt because insufficient power
)

type Structure struct {
	ID     int    // Structure type ID
	Name   string // Name for frontend printing
	Status SStatus

	Population    int   // + for provide, - for occupy
	Money         int   // + for produce, - for consume
	Power         int   // + for generate, - for consume
	PopulationCap int   // How many population can increase for player
	BuildTime     int64 // How many time before building finish

	Cost int // Money required for build
	// Upgrade cost: 1->2 = 1 * Cost
	//               2->3 = 2 * Cost
	//               3->4 = 4 * Cost
	//               4->5 = 8 * Cost

	Level    int // Building's current level
	MaxLevel int

	Chunk util.Point
	Pos   util.Point
	Size  util.Size

	Terrain    int   // vaild construct terrain
	UpdateTime int64 // Unix time
}
