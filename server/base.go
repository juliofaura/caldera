package server

import (
	//"encoding/gob"

	"flag"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/juliofaura/webutil"
)

///////////////////////////////////////////////////
// Constants (some in fact defined as global variables) and types
///////////////////////////////////////////////////

const (
	WEB_PATH               = "./web/"
	HEADER_TEMPLATE_FILE   = "header.html"
	ERROR_TEMPLATE_FILE    = "error.html"
	BACKGROUNDPICSDIR      = "(not used)"
	SESSIONNAMEPREFIX      = "calderaWebSession"
	SESSIONSTORENAMEPREFIX = "calderaWebCookiestore2345234xjhkh"
	SESSIONALERTSPREFIX    = "calderaWebPendingAlerts"
)

var (
	WEBPORT           string = "8050"
	HEADER_PAGE_TITLE string = "Header page title"
)

var (
	SESSIONNAME      string
	SESSIONSTORENAME string
	SESSIONALERTS    string
)

var templates = template.Must(template.ParseFiles(
	WEB_PATH+"caldera.html",
	WEB_PATH+"theme.html",
))

var consoleUsers = map[string]webutil.ConsoleUserT{
	//TO DO: put this in the DB or at least into a file
	"admin": {"admin", "1234", true},
}

func StartWeb() {

	SESSIONNAME = SESSIONNAMEPREFIX + WEBPORT
	SESSIONSTORENAME = SESSIONSTORENAMEPREFIX + WEBPORT
	SESSIONALERTS = SESSIONALERTSPREFIX + WEBPORT

	flag.Parse()

	webutil.Init(
		WEB_PATH,
		HEADER_PAGE_TITLE,
		HEADER_TEMPLATE_FILE,
		ERROR_TEMPLATE_FILE,
		// BACKGROUNDPICSDIR,
		// data.PICSPREFIX,
		"notUsed",
		SESSIONNAME,
		SESSIONSTORENAME,
		SESSIONALERTS,
		consoleUsers,
	)
	http.Handle("/", http.HandlerFunc(HandleCaldera))
	http.Handle("/caldera", http.HandlerFunc(HandleCaldera))
	http.Handle("/poweron", http.HandlerFunc(HandlePowerOn))
	http.Handle("/poweroff", http.HandlerFunc(HandlePowerOff))
	http.Handle("/thermostaton", http.HandlerFunc(HandleThermostatOn))
	http.Handle("/thermostatoff", http.HandlerFunc(HandleThermostatOff))
	http.Handle("/changetemp", http.HandlerFunc(HandleChangeTemp))
	// http.Handle("/gasoleo", http.HandlerFunc(HandleGasoleo))
	// http.Handle("/temperatura", http.HandlerFunc(HandleTemperatura))
	http.Handle("/theme", http.HandlerFunc(HandleTheme))
	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir(WEB_PATH+"resources"))))
	//http.Handle("/local_resources/", http.StripPrefix("/local_resources/", http.FileServer(http.Dir("./local_resources"))))
	go func() {
		addr := flag.String("addr", ":"+WEBPORT, "http service address")
		err := http.ListenAndServe(*addr, context.ClearHandler(http.DefaultServeMux))
		if err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()
}

func HandleTheme(w http.ResponseWriter, req *http.Request) {
	templates.ExecuteTemplate(w, "theme.html", "")
}
