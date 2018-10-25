package player

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "encoding/json"
)

type gitlab_user struct {
    Id int
    Username string
}

// Login with GitLab access token
func Login(token string) (string, bool) {
    client := &http.Client{}

    //TODO: use config file for hostname
    req, err := http.NewRequest("GET", "https://pd2a.imslab.org/gitlab/api/v4/user", nil)
    if err != nil {
        fmt.Println(err.Error())
    }

    // TODO: change to "Access-Token" for OAuth (current use private token for test)
    req.Header.Set("Private-Token", token)

    rsp, err := client.Do(req)
    defer rsp.Body.Close()

    body, err := ioutil.ReadAll(rsp.Body)
    if err != nil {
        fmt.Println(err.Error())
    }

    var user gitlab_user
    json.Unmarshal(body, &user)
    return user.Username, true
}
