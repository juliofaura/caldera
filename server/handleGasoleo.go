package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/juliofaura/caldera/data"
	"github.com/juliofaura/oilmeter/files"
	"github.com/juliofaura/webutil"
)

func HandleGasoleo(w http.ResponseWriter, req *http.Request) {
	data.M.Lock()
	defer data.M.Unlock()

	// webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Success!")
	// webutil.Reload(w, req, "/")

	datums, err := files.ReadDataFile(files.DataFile)
	if err != nil {
		webutil.PushAlertf(w, req, webutil.ALERT_DANGER, "Error leyendo los datos del gasoleo")
		webutil.Reload(w, req, "/caldera")
		return
	}
	avgFile, err := os.Open(files.AverageFile)
	if err != nil {
		return
	}
	defer avgFile.Close()
	var avg float64
	n, err := fmt.Fscanln(avgFile, &avg)
	if n == 0 || err != nil {
		webutil.PushAlertf(w, req, webutil.ALERT_DANGER, "Error leyendo los datos del consumo medio de gasoleo")
		webutil.Reload(w, req, "/caldera")
		return
	}

	passdata := map[string]interface{}{
		"liters": fmt.Sprintf("%.1f", datums[len(datums)-1].Liters),
		"avg":    avg,
		"datums": datums,
	}
	webutil.PlaceHeader(w, req)
	templates.ExecuteTemplate(w, "gasoleo.html", passdata)
}
