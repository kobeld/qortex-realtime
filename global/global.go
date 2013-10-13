package global

import (
	"github.com/kobeld/qortex-realtime/configs"
	"github.com/sunfmin/mgodb"
)

func IsQortexSupportDB(db *mgodb.Database) bool {
	return db.DatabaseName == configs.QORTEX_SUPPORT_DB_NAME
}
