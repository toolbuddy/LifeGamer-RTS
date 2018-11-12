package player

import (
    "util"
)

type Player struct {
    Human int
    Money int
    Power int
    Home util.Point // spawn point
    Initialized bool
    UpdateTime int64
}
