package world

type TerrainType int

// Null value is 0
const (
	Null TerrainType = iota
)

// Use shift for terrain value
const (
	Desert TerrainType = 1 << iota
	Grass
	Forest
	Sea
	River
	Snow
	Coast
	Bank
	Lava
	Volcano
)

// Check if a terrain accepts struct's buildable terrain
func (terrain TerrainType) Accepts(terrains int) bool {
	if int(terrain)&terrains != 0 {
		return true
	}

	return false
}
