package reqres

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type User struct {
	Id        int
	Email     string
	FirstName string
	LastName  string
	Avatar    string
}

type UserCreateResponse struct {
	Id        string `json:"id"`
	CreatedAt string `json:"createdAt"`
}

type UserGetResponse struct {
	Data struct {
		Id        int    `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name,omitempty"`
		LastName  string `json:"last_name,omitempty"`
		Avatar    string `json:"avatar,omitempty"`
	} `json:"data"`
	Support struct{} `json:"support,omitempty"`
}

const (
	notInitialized    = 0
	httpPostSuccess   = 201
	httpGetSuccess    = 200
	httpDeleteSuccess = 204
	httpPatchSuccess  = 204
	ctrlFinalizer     = "users.reqres.in/v1alpha1"
	usersApi          = "/api/users/"
)

func (c *Client) CreateUser(user User) (*User, error) {
	postBody, _ := json.Marshal(map[string]string{
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	})
	body := bytes.NewBuffer(postBody)
	url := c.HostUrl + usersApi
	httpReq, _ := http.NewRequest("POST", url, body)
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != httpPostSuccess {
		return nil, fmt.Errorf("http status: %d", res.StatusCode)
	}
	var response UserCreateResponse
	resBody, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(resBody, &response)
	id, _ := strconv.Atoi(response.Id)
	return &User{Id: id}, nil
}

func (c *Client) UpdateUser(user User) error {
	postBody, _ := json.Marshal(map[string]string{
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	})
	api := usersApi + strconv.Itoa(user.Id)
	url := c.HostUrl + api
	body := bytes.NewBuffer(postBody)
	httpReq, _ := http.NewRequest("PATCH", url, body)
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error making http request")
	}
	defer res.Body.Close()
	if res.StatusCode != httpPatchSuccess {
		return fmt.Errorf("http status: %d", res.StatusCode)
	}
	return nil
}

func (c *Client) GetUser(id int) (*User, error) {
	api := usersApi + strconv.Itoa(id)
	url := c.HostUrl + api
	httpReq, _ := http.NewRequest("GET", url, nil)
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error making http request")
	}
	defer res.Body.Close()
	if res.StatusCode != httpGetSuccess {
		return nil, fmt.Errorf("http status: %d", res.StatusCode)
	}
	resBody, _ := ioutil.ReadAll(res.Body)
	var userGetResponse UserGetResponse
	json.Unmarshal(resBody, &userGetResponse)
	return &User{
		Id:        userGetResponse.Data.Id,
		Email:     userGetResponse.Data.Email,
		FirstName: userGetResponse.Data.FirstName,
		LastName:  userGetResponse.Data.LastName,
		Avatar:    userGetResponse.Data.Avatar,
	}, nil
}

func (c *Client) DeleteUser(id int) (bool, error) {
	api := usersApi + strconv.Itoa(id)
	url := c.HostUrl + api
	httpReq, _ := http.NewRequest("DELETE", url, nil)
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("error making http request")
	}
	defer res.Body.Close()
	if res.StatusCode != httpDeleteSuccess {
		return false, fmt.Errorf("http status: %d", res.StatusCode)
	}
	return true, nil
}
