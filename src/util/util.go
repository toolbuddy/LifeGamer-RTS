package util

import "fmt"

type Point struct {
    X, Y int
}

type Size struct {
    W, H uint
}

func (point Point) String() string {
    return fmt.Sprintf("%d,%d", point.X, point.Y)
}

func (point Point) Up() Point {
    return Point { point.X, point.Y - 1 }
}

func (point Point) Down() Point {
    return Point { point.X, point.Y + 1 }
}

func (point Point) Left() Point {
    return Point { point.X - 1, point.Y }
}

func (point Point) Right() Point {
    return Point { point.X + 1, point.Y }
}

// Returns a list contains all points between from and to
func InRange(from Point, to Point) (res []Point) {
    for p := from; p.Y <= to.Y; p = p.Down() {
        for q := p; q.X <= to.X; q = q.Right() {
            res = append(res, q)
        }
    }

    return
}

// Return a list contains all points in size range (from is left-upper point)
func InSizeRange(from Point, size Size) (res []Point) {
    for dy := 0; uint(dy) < size.H; dy++ {
        for dx := 0; uint(dx) < size.W; dx++ {
            res = append(res, Point { X: from.X + dx, Y: from.Y + dy })
        }
    }

    return
}
