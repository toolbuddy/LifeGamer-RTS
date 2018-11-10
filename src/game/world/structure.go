package world

import (
    "util"
)

type Structure struct {
    Hp      int
    Human   int             // + for provide, - for occupy
    Money   int             // + for produce, - for consume
    Pos     util.Point
    Power   int             // + for generate, - for consume
    Size    util.Size
    Terrain []TerrainType   // vaild construct terrain
}
