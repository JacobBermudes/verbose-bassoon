package moolah

import "os"

var APIKEY = os.Getenv("FFIO_APIKEY")

type ffio struct {
	Client *Client
}
