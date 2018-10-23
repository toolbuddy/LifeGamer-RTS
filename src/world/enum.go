package world

type TerrainType uint

const (
    Null TerrainType = 1 << iota
    Desert
    Grassland
    Forest
    Sea
    River
    Snow
    Seacoast
    Riverbank
    Volcano
)
