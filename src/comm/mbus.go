package comm

import (
    "errors"
)

var generator (func() int)
var chans map[string]chan []byte

func init() {
    // cid generator
    generator = func() (func() int) {
        var i int = -1
        return func() int {
            i++
            return i
        }
    }()

    chans = make(map[string]chan []byte)
}

type MBusNode struct {
    cid int
    name string
    ReaderChan <-chan []byte
}

func NewMBusNode(name string) (node *MBusNode, err error) {
    // Check node not exist
    if _, ok := chans[name]; ok {
        err = errors.New("Node already exist")
        return
    }

    new_id := generator()
    chans[name] = make(chan []byte, 256)

    node = &MBusNode { new_id, name, chans[name] }

    return
}

// Read single message from MBus
func (c MBusNode) Read() []byte {
    select {
    case msg := <-chans[c.name]:
        return msg
    default:
        return nil
    }
}

// Write message to node 'dst', return false if no such node
func (c MBusNode) Write(dst string, msg []byte) (ok bool) {
    dstchan, ok := chans[dst]

    if ok {
        dstchan <- msg
    }

    return
}
