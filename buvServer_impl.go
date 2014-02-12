package buv

import (
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

const (
	LOG_PRINTLN = 0
	LOG_FATAL   = 1
)

func (b *BuvServer) renderTemplate(w http.ResponseWriter, tmpl string, p interface{}) {
	err := b.myTemplates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		b.logger.Println("Error renderTemplate: " + err.Error())
	}
}

func (b *BuvServer) renderer(fn BuvHandleFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, b.renderTemplate, b.logger)
	}
}

func (b *BuvServer) assetHandler(assetFolder string) http.HandlerFunc {
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
