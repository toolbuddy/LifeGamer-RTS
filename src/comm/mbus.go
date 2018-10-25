package comm

var generator (func() int)
var chans map[string]chan string

func init() {
    generator = func() (func() int) {
        var i int = -1
        return func() int {
            i++
            return i
        }
    }()

    chans = make(map[string]chan string)
}

type MBusNode struct {
    cid int
    name string
}

func NewMBusNode(name string) MBusNode {
    new_id := generator()
    chans[name] = make(chan string, 100)

    return MBusNode { new_id, name }
}

func (c MBusNode) Get() (string, bool) {
    select {
    case msg := <-chans[c.name]:
        return msg, true
    default:
        return "", false
    }
}

func (c MBusNode) Put(dst string, msg string) (ok bool) {
    dstchan, ok := chans[dst]

    if ok {
        dstchan <- msg
    }

    return ok
}
