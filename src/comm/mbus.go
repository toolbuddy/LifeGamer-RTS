package comm

import (
    "errors"
)

var chans map[string] chan MessageWrapper

func init() {
    chans = make(map[string] chan MessageWrapper)
}

type MBusNode struct {
    name string
    ReaderChan <-chan MessageWrapper
}

func NewMBusNode(name string) (node *MBusNode, err error) {
    // Check node not exist
    if _, ok := chans[name]; ok {
        err = errors.New("Node already exist")
        return
    }

    chans[name] = make(chan MessageWrapper, 256)
    node = &MBusNode { name, chans[name] }
    return
}

// Read single message from MBus
func (c MBusNode) Read() (msg MessageWrapper, ok bool) {
    select {
    case msg, ok = <-chans[c.name]:
        return
    default:
        ok = false
        return
    }
}

// Write message to node 'dst', return false if no such node
func (c MBusNode) Write(dst string, msg MessageWrapper) (ok bool) {
    dstchan, ok := chans[dst]
    if ok {
        dstchan <- msg
    }

    return
}
