package comm

import (
    "fmt"
    "encoding/json"
    "os"
)

/*
 * Message type for server/client communication.
 *
 * XXX: Remember to add type name to method `String` after you add a new type.
 *      If the new type is added at the end of const block,
 *      change the end condition of loop in method `MsgType2Json` also.
 *
 * XXX: Please write type name with the same order of constants!!!
 */
type MsgType uint

const (
    // Request from client
    LoginRequest MsgType = iota
    PlayerDataRequest
    MapDataRequest
    LogoutRequest

    // Request from server
    HomePointRequest

    // Response from server
    LoginResponse
    PlayerDataResponse
    MapDataResponse

    // Response from client
    HomePointResponse
)

func (mtype MsgType) String() string {
    return [...]string {
        "LoginRequest",
        "PlayerDataRequest",
        "MapDataRequest",
        "LogoutRequest",

        "HomePointRequest",

        "LoginResponse",
        "PlayerDataResponse",
        "MapDataResponse",

        "HomePointResponse",
    }[mtype]
}

// Call this function to generate message type json file for client
func MsgType2Json() (err error) {
    m := make(map[string]MsgType)

    for i := MsgType(0); i <= HomePointResponse; i++ {
        m[i.String()] = i
    }

    b, err := json.Marshal(m)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        return
    }

    fp, err := os.OpenFile("message_type.json", os.O_CREATE | os.O_WRONLY, 0644)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        return
    }

    defer fp.Close()

    fmt.Fprintln(fp, string(b))

    return
}

// Data container for server/client communication
type Payload struct {
    Msg_type MsgType
    Username string
    Message  string
}
