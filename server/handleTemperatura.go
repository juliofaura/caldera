package server

import (
	"net/http"

	"github.com/juliofaura/webutil"
)

func HandleTemperatura(w http.ResponseWriter, req *http.Request) {

	// webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Success!")
	// webutil.Reload(w, req, "/")

	passdata := map[string]interface{}{
		"etc": "etc",
	}
	webutil.PlaceHeader(w, req)
	templates.ExecuteTemplate(w, "temperatura.html", passdata)
}
