package player

import (
    "util"
)

type Player struct {
    Id string       // student id
    Human int
    Money int
    Power int
    Home util.Point // spawn point
}
