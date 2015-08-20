package main

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
)

func TestStatusRequest(t *testing.T) {
	go main()

	resp, err := http.Get("http://localhost:8080/status")
	log.Println(resp)
	defer resp.Body.Close()
	assert.NoError(t, err)

	body, ioerr := ioutil.ReadAll(resp.Body)
	assert.NoError(t, ioerr)
	assert.Equal(t, "OK", string(body), "resp body should match")
	assert.Equal(t, "200 OK", resp.Status, "should get a 200")
}
