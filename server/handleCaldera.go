package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/juliofaura/caldera/data"
	"github.com/juliofaura/webutil"
)

func HandleCaldera(w http.ResponseWriter, req *http.Request) {
	data.M.Lock()
	defer data.M.Unlock()

	// webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Success!")
	// webutil.Reload(w, req, "/")

	data.ReadPower()
	data.ReadHeat()
	data.ReadTemp()

	if data.PowerOn != data.PowerReading {
		msg := "Error! - la caldera está "
		if data.PowerOn {
			msg += "encendida"
		} else {
			msg += "apagada"
		}
		msg += " y debería estar "
		if data.PowerReading {
			msg += "encendida"
		} else {
			msg += "apgada"
		}
		webutil.PushAlertf(w, req, webutil.ALERT_DANGER, msg)
	}

	if data.HeatOn != data.HeatReading {
		msg := "Error! - el calentador está "
		if data.HeatOn {
			msg += "encendido"
		} else {
			msg += "apagado"
		}
		msg += " y debería estar "
		if data.HeatReading {
			msg += "encendido"
		} else {
			msg += "apagado"
		}
		webutil.PushAlertf(w, req, webutil.ALERT_DANGER, msg)
	}

	if data.ErrorInTemp {
		msg := "Error! - error al medir la temperatura del sensor (" + data.Sensor + ")"
		webutil.PushAlertf(w, req, webutil.ALERT_DANGER, msg)

	}

	passdata := map[string]interface{}{
		"power":       data.PowerOn,
		"thermostat":  data.ThermostatOn,
		"heater":      data.HeatOn,
		"sensor":      data.Sensor,
		"temperature": data.CurrentTemp,
		"targettemp":  data.TargetTemp,
	}
	webutil.PlaceHeader(w, req)
	templates.ExecuteTemplate(w, "caldera.html", passdata)
}

func HandlePowerOn(w http.ResponseWriter, req *http.Request) {
	data.M.Lock()
	defer data.M.Unlock()
	data.SetPower(data.ON)
	data.WriteConfig()
	webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Encendida la caldera")
	webutil.Reload(w, req, "/caldera")
}

func HandlePowerOff(w http.ResponseWriter, req *http.Request) {
	data.M.Lock()
	defer data.M.Unlock()
	data.SetPower(data.OFF)
	data.WriteConfig()
	webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Apagada la caldera")
	webutil.Reload(w, req, "/caldera")
}

func HandleThermostatOn(w http.ResponseWriter, req *http.Request) {
	data.M.Lock()
	defer data.M.Unlock()
	data.ThermostatOn = true
	data.WriteConfig()
	webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Activado el termostato")
	webutil.Reload(w, req, "/caldera")
}

func HandleThermostatOff(w http.ResponseWriter, req *http.Request) {
	data.M.Lock()
	defer data.M.Unlock()
	data.ThermostatOn = false
	data.WriteConfig()
	webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Desactivado el termostato")
	webutil.Reload(w, req, "/caldera")
}

func HandleChangeTemp(w http.ResponseWriter, req *http.Request) {
	data.M.Lock()
	defer data.M.Unlock()
	req.ParseForm()
	newTempA, oknewtemp := req.Form["newtemp"]
	if !oknewtemp {
		webutil.PushAlert(w, req, webutil.ALERT_DANGER, "Error al cambiar la temperatura objetivo, faltan datos")
		webutil.Reload(w, req, "/")
		return
	}
	newTemp, err := strconv.ParseFloat(newTempA[0], 64)
	if err != nil {
		webutil.PushAlertf(w, req, webutil.ALERT_DANGER, "Temperatura incorrecta ("+newTempA[0]+")")
		webutil.Reload(w, req, "/")
		return
	}
	data.TargetTemp = newTemp
	data.WriteConfig()
	webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Cambiada la temperatura objetivo a "+fmt.Sprint(newTemp))
	webutil.Reload(w, req, "/caldera")
}
