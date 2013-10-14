package global

import (
	"github.com/kobeld/qortex-realtime/configs"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/qortex/utils"
	"html/template"
	"time"
)

func IsQortexSupportDB(db *mgodb.Database) bool {
	return db.DatabaseName == configs.QORTEX_SUPPORT_DB_NAME
}

// Time/Date utils
func ConvertStrToTime(timeStr string) (r time.Time) {
	if timeStr != "" {
		r = utils.UnixNanoToTime(timeStr)
	} else {
		r = time.Now()
	}

	return
}

func StringToHtml(str string) template.HTML {
	return (template.HTML)(template.HTMLEscapeString(str))
}
