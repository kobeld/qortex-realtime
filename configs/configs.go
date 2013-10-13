package configs

import (
	"time"
)

var (
	WSPort         = ":5055"
	NsqLookupAdddr = "localhost:4161"
)

const (
	ONLINE_USER_CLOSE_DURATION = 10 * time.Second
	QORTEX_SUPPORT_DB_NAME     = "qortex_global"
)
