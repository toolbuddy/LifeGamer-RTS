package comm

import (
    "errors"
)

var generator (func() int)
var chans map[string]chan interface{}

func init() {
    // cid generator
    generator = func() (func() int) {
        var i int = -1
        return func() int {
            i++
            return i
        }
    }()

    chans = make(map[string]chan interface{})
}

type MBusNode struct {
    cid int
    name string
    ReaderChan <-chan interface{}
}

func NewMBusNode(name string) (MBusNode, error) {
    var newnode MBusNode

    // Check node not exist
    _, ok := chans[name]
    if ok {
        return newnode, errors.New("Node already exist")
    }

    new_id := generator()
    chans[name] = make(chan interface{}, 256)

    newnode = MBusNode { new_id, name, chans[name] }

    return newnode, nil
}

// Read single message from MBus
func (c MBusNode) Read() interface{} {
    select {
    case msg := <-chans[c.name]:
        return msg
    default:
        return nil
    }
}

// Write message to node 'dst', return false if no such node
func (c MBusNode) Write(dst string, msg interface{}) (ok bool) {
    dstchan, ok := chans[dst]

    if ok {
        dstchan <- msg
    }

    return
}
