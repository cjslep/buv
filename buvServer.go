package buv

import (
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
	"os"
	"fmt"
	"time"
	"net"
)

type TemplateRenderer func (w http.ResponseWriter, tmpl string, p interface{})
type BevHandleFunc func (w http.ResponseWriter, r *http.Request, f TemplateRenderer, l TimeLogger)

func TrackElapsed(start time.Time, name string) string {
    elapsed := time.Since(start)
    return fmt.Sprintf("%s took %s", name, elapsed)
}

type WebServer interface {
	Start(domain string, templateFiles []string, address, cssFolder, jsFolder string, muxToHandler map[string]BevHandleFunc, notFoundHandler BevHandleFunc) error
	Shutdown()
}

type TimeLogger interface {
	Println(output string)
	Fatal(output string)
}

func NewBuvServer(fileLog, dirLog string, filePerms, dirPerms os.FileMode) (w WebServer, e error) {
	logger, err := newBasicTimeLogger(fileLog, dirLog, filePerms, dirPerms)
	if err != nil { return nil, err }
	server := buvServer{nil, make(map[string]BevHandleFunc), logger, nil, nil}
	logger.Println("Successfully made buvServer!")
	return &server, nil
}

func (b *buvServer) Start(domain string, templateFiles []string, address, cssFolder, jsFolder string, muxToHandler map[string]BevHandleFunc, notFoundHandler BevHandleFunc) error {
	defer b.logger.Println(TrackElapsed(time.Now(), "*Server Startup*"))
	b.logger.Println("Begin *Server Startup*")
	b.logger.Println("Parsing template files...")
	b.myTemplates = template.Must(template.ParseFiles(templateFiles...))
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
	s.HandleFunc(cssFolder + "{asset:[a-z]+(.css)}", b.assetHandler(cssFolder))
    b.logger.Println("JS handler using folder: " + jsFolder)
    b.logger.Println("JS handler for: " + jsFolder + "{asset:[a-z]+(.js)}")
	s.HandleFunc(jsFolder + "{asset:[a-z]+(.js)}", b.assetHandler(jsFolder))
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
	go func(l net.Listener, ch chan <-bool) {
		b.logger.Println("Begin serving on listener with address: " + l.Addr().String())
		http.Serve(l, nil)
		b.logger.Println("Ending Serve. Sending shutdown notification to channel")
		ch <- true
	}(b.listener, b.servNotifier)
	return nil
}

func (b *buvServer) Shutdown() {
	defer b.logger.Println(TrackElapsed(time.Now(), "*Server Shutdown*"))
	b.logger.Println("Begin *Server Shutdown*")
	b.logger.Println("Closing the listener.")
	b.listener.Close()
	b.logger.Println("Waiting for shutdown notification.")
	<-b.servNotifier
}