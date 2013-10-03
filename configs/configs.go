package configs

import (
	"time"
)

var (
	WSPort         = ":5055"
	NsqLookupAdddr = "localhost:4161"
)

var (
	ONLINE_USER_CLOSE_DURATION = 10 * time.Second
)
