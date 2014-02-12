// Package buv is a web server dedicated to being a slave to Web 2.0, serving up web pages
// and parsing templates and providing a logger to client handling code. It allows
// a client to specify gorilla-style mux's to specific handlers
package buv

import (
	"bitbucket.org/cjslep/dailyLogger"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"net"
	"net/http"
	"os"
	"time"
)

// A BuvServer is a http server that is able to gracefully start and terminate connections as it starts up
// and shuts down, in addition to handling template execution, logging details, and mapping muxes to client
// code handlers as configured.
type BuvServer struct {
	myTemplates  *template.Template
	handlers     map[string]BuvHandleFunc
	logger       dailyLogger.TimeLogger
	listener     net.Listener
	servNotifier chan bool
}

// BuvHandleFunc is the function clients must use when handling requests. It provides access to the web logger
// in addition to a template renderer function that enables the handler to display a particular template if desired.
type BuvHandleFunc func(w http.ResponseWriter, r *http.Request, f TemplateRenderer, l dailyLogger.TimeLogger)

// TemplateRenderer functions accept a template name and a data structure to execute the template on. Its purpose
// is to be used by client code in order to send http data to a user.
type TemplateRenderer func(w http.ResponseWriter, tmpl string, p interface{})

// Convenience function to allow time tracking. Best used when deferred.
func TrackElapsed(start time.Time, name string) string {
	elapsed := time.Since(start)
	return fmt.Sprintf("%s took %s", name, elapsed)
}

// TODO: Eliminate this interface
type WebServer interface {
	Start(domain string, templateFiles []string, address, cssFolder, jsFolder string, muxToHandler map[string]BuvHandleFunc, notFoundHandler BuvHandleFunc) error
	Shutdown()
}

// Creates a new Buv web server, using fileLog as the name of the file for logging, the directory for
// storing the logs, and the file and directory permissions for the logger.
func NewBuvServer(fileLog, dirLog string, filePerms, dirPerms os.FileMode) (w *BuvServer, e error) {
	logger, err := dailyLogger.NewBasicTimeLogger(fileLog, dirLog, filePerms, dirPerms)
	if err != nil {
		return nil, err
	}
	server := BuvServer{nil, make(map[string]BuvHandleFunc), logger, nil, nil}
	logger.Println("Successfully made buvServer!")
	return &server, nil
}

// Starts up the web service, using the specified domain, template files, port address, css & javascript asset folders,
// handler map, and default handler for invalid URIs.
func (b *BuvServer) Start(domain string, templateFiles []string, address, cssFolder, jsFolder string, muxToHandler map[string]BuvHandleFunc, notFoundHandler BuvHandleFunc) error {
	defer b.logger.Println(TrackElapsed(time.Now(), "*Server Startup*"))
	b.logger.Println("Begin *Server Startup*")
	b.logger.Println("Parsing template files...")
	var err error
	b.myTemplates, err = template.ParseFiles(templateFiles...)
	if err != nil {
		b.logger.Println(err.Error())
		panic(err.Error())
	}
	b.logger.Println("Done parsing template files!")

	r := mux.NewRouter()
	r.NotFoundHandler = b.renderer(notFoundHandler)
	var s *mux.Router
	if len(domain) == 0 {
		b.logger.Println("Using localhost as the host.")
		s = r.Host("localhost").Subrouter()
	} else {
		b.logger.Println("Using \"" + domain + "\" as the host.")
		s = r.Host(domain).Subrouter()
	}
	for key, value := range muxToHandler {
		b.logger.Println("Adding HandleFunc for: " + key)
		b.handlers[key] = value
		s.HandleFunc(key, b.renderer(value))
	}
	b.logger.Println("CSS handler using folder: " + cssFolder)
	b.logger.Println("CSS handler for: " + cssFolder + "{asset:[a-z]+(.css)}")
	s.HandleFunc(cssFolder+"{asset:[a-z]+(.css)}", b.assetHandler(cssFolder))
	b.logger.Println("JS handler using folder: " + jsFolder)
	b.logger.Println("JS handler for: " + jsFolder + "{asset:[a-z]+(.js)}")
	s.HandleFunc(jsFolder+"{asset:[a-z]+(.js)}", b.assetHandler(jsFolder))
	http.Handle("/", r)
	b.logger.Println("Finished building handlers.")

	b.logger.Println("Creating listener on address " + address)
	list, err := net.Listen("tcp", address)
	b.listener = list
	if err != nil {
		b.logger.Fatal("Error: " + err.Error())
	}

	b.logger.Println("Creating channel for shutdown notification.")
	b.servNotifier = make(chan bool)
	go func(l net.Listener, ch chan<- bool) {
		b.logger.Println("Begin serving on listener with address: " + l.Addr().String())
		http.Serve(l, nil)
		b.logger.Println("Ending Serve. Sending shutdown notification to channel")
		ch <- true
	}(b.listener, b.servNotifier)
	return nil
}

// Gracefully shuts down the Buv web server and terminates connections.
func (b *BuvServer) Shutdown() {
	defer b.logger.Println(TrackElapsed(time.Now(), "*Server Shutdown*"))
	b.logger.Println("Begin *Server Shutdown*")
	b.logger.Println("Closing the listener.")
	b.listener.Close()
	b.logger.Println("Waiting for shutdown notification.")
	<-b.servNotifier
}
