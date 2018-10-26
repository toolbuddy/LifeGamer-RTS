package comm

import (
    "errors"
    "net/http"
    "io/ioutil"
    "encoding/json"
)

type gitlab_user struct {
    Id int
    Username string
    Message string
}

// Login with GitLab access token
func Login(token string) (string, error) {
    client := &http.Client{}

    //TODO: use config file for hostname
    req, err := http.NewRequest("GET", "https://pd2a.imslab.org/gitlab/api/v4/user", nil)
    if err != nil {
        return "", err
    }

    // TODO: change to "Access-Token" for OAuth (current use private token for test)
    req.Header.Set("Private-Token", token)

    rsp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer rsp.Body.Close()

    body, err := ioutil.ReadAll(rsp.Body)
    if err != nil {
        return "", err
    }

    var user gitlab_user
    json.Unmarshal(body, &user)

    // Error from GitLab server
    if user.Message != "" {
        return "", errors.New(user.Message)
    }

    return user.Username, nil
}