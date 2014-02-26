package buv

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"net/http"
	"os"
)

const (
	LOG_PRINTLN = 0
	LOG_FATAL   = 1
)

func redirectOrHandler(b HandlerFunction, redirs ...Redirector) HandlerFunction {
	return func (data *HandlerData) {
		//data.Println("Request: " + data.String())
		for _, redirFunc := range redirs {
			if redirFunc(data) {
				return
			}
		}
		b(data)
	}
}

func (b *Server) getSession(request *http.Request, sessionName string) *sessions.Session {
	sess, err := b.cookieStore.Get(request, sessionName)
	if err != nil {
		b.logger.Println(err.Error())
		return nil
	}
	return sess
}

func (b *Server) saveSession(r *http.Request, w http.ResponseWriter, session *sessions.Session) {
	err := session.Save(r, w)
	if err != nil {
		b.logger.Println(err.Error())
	}
}

func (b *Server) handler(fn HandlerFunction) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		temp := HandlerData{w, r, b}
		fn(&temp)
	}
}

func (b *Server) assetHandler(assetFolder string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.logger.Println("Handling asset: " + r.URL.Path)
		vars := mux.Vars(r)
		file, err := os.Open("." + r.URL.Path)
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
