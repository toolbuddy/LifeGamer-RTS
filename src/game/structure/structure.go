package structure

import (
    "util"
    "game/world"
)

type Structure struct {
    Hp int
    Human int                   // + for provide, - for occupy
    Money int                   // + for produce, - for consume
    Pos util.Point
    Power int                   // + for generate, - for consume
    Size util.Size
    Terrain []world.TerrainType // vaild construct terrain
}
