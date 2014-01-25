package buv

import (
	"github.com/gorilla/mux"
	"log"
	"time"
	"os"
	"strings"
	"io"
	"net"
	"net/http"
	"html/template"
)

const (
	LOG_PRINTLN = 0
	LOG_FATAL = 1
)

type basicTimeLogger struct {
	logFile string
	logFilePerms os.FileMode
	logDir string
	currTime time.Time
	quitChan chan bool
	logChan chan logMessage
}

type closableWriter interface {
	io.Writer
	Close() error
}

type logMessage struct {
	MsgType int
	Message []byte
}

func (b *basicTimeLogger) formatTimeString(timeString string) string {
	return strings.Replace(strings.Replace(timeString, " ", "_", -1), ":", "-", -1)
}

func (b *basicTimeLogger) updateIfNewDay() {
	if b.currTime.Day() != time.Now().Local().Day() {
		b.currTime = time.Now().Local()
		b.setNewLogger()
	}
}

func (b *basicTimeLogger) setNewLogger() {
	if (b.quitChan != nil) {
		b.quitChan <- true
		close(b.quitChan)
		close(b.logChan)
	}
	file, err := os.OpenFile(b.logDir + b.formatTimeString(b.currTime.Format(time.Stamp)) + b.logFile + ".txt", os.O_APPEND | os.O_CREATE | os.O_RDWR, b.logFilePerms)
	if err != nil { return }
	b.quitChan = make(chan bool)
	b.logChan = make(chan logMessage)
	go func(f closableWriter, toLog <- chan logMessage, quit <- chan bool) {
		defer f.Close()
		logger := log.New(f, "", log.LstdFlags | log.Lshortfile)
		for {
			select {
				case msg := <- toLog:
					if (msg.MsgType == LOG_PRINTLN) {
						logger.Println(string(msg.Message))
					} else if (msg.MsgType == LOG_FATAL) {
						logger.Fatal(string(msg.Message))
					}
				case <- quit:
					return
			}
		}
	}(file, b.logChan, b.quitChan)
	b.logChan <- logMessage{LOG_PRINTLN, []byte("Logging goroutine successfully launched!")}
}

func (b *basicTimeLogger) logMessage(msgType int, msg string) {
	b.updateIfNewDay()
	b.logChan <- logMessage{msgType, []byte(msg)}
}

func (b *basicTimeLogger) Println(output string) {
	b.logMessage(LOG_PRINTLN, output)
}

func (b *basicTimeLogger) Fatal(output string) {
	b.logMessage(LOG_FATAL, output)
}

func newBasicTimeLogger(fileLog, dirLog string, filePerms, dirPerms os.FileMode) (t TimeLogger, err error) {
	temp := basicTimeLogger{fileLog, filePerms, dirLog, time.Now().Local(), nil, nil}
	temp.setNewLogger()
	err = os.MkdirAll(dirLog, dirPerms)
	if err != nil { return nil, err }
	return &temp, nil
}

type buvServer struct {
	myTemplates *template.Template
	handlers map[string]BevHandleFunc
	logger TimeLogger
	listener net.Listener
	servNotifier chan bool
}

func (b *buvServer) renderTemplate(w http.ResponseWriter, tmpl string, p interface{}) {
	err := b.myTemplates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (b *buvServer) renderer(fn BevHandleFunc) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		fn(w, r, b.renderTemplate, b.logger)
	}
}

func (b *buvServer) assetHandler(assetFolder string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.logger.Println("Handling asset: " + r.URL.Path)
		vars := mux.Vars(r)
		file, err := os.Open("."+r.URL.Path)
		defer file.Close()
		if err != nil {
				http.NotFound(w, r)
				return
		}
		stat, err := file.Stat()
		if err != nil {
				http.NotFound(w, r)
				return
		}
		http.ServeContent(w, r, vars["asset"], stat.ModTime(), file)
	}
}