package comm

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"config"
)

type gitlab_user struct {
	Id       int
	Username string
	Message  string
}

// Login with GitLab token
func Login(token string, token_type string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", config.Hostname+"/gitlab/api/v4/user", nil)
	if err != nil {
		return "", err
	}

	// Set access token type
	q := req.URL.Query()
	q.Add(token_type, token)
	req.URL.RawQuery = q.Encode()

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
