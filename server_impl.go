package buv

/*
	This file is a part of Buv
	Copyright (C) 2014  Cory J. Slep

    Buv is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    Buv is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

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

// Convenience function to allow time tracking when debugging. Best used when deferred.
func trackElapsed(start time.Time, name string) string {
	elapsed := time.Since(start)
	return fmt.Sprintf("%s took %s", name, elapsed)
}