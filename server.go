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
	"bitbucket.org/cjslep/dailyLogger"
	"bitbucket.org/cjslep/goTem"
	"gopkg.in/v1/yaml"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Server is a http server that is able to gracefully start and terminate TCP-over-IP
// connections as it starts up and shuts down. In brief it handles:
//   - Template execution & template dependencies
//   - One-Log-File-A-Day Logging
//   - Rich mux mapping to client handlers, guarded by redirectors
//   - Mapping a path to a filetype to serve up specific assets
//   - Constructing data-driven URLs
//   - Handling session values and flash messages
//   - New, re-used, or rotated secure cookie keys
// A Server does not directly interact with the handlers, instead it exposes a limited
// subset of its interface through a HandlerData that contains additional request
// information beyond what the sole Server provides.
type Server struct {
	templateManager *goTem.HTMLTemplateWatcher
	handlers        map[string]HandlerFunction
	logger          *dailyLogger.DailyLogger
	listener        net.Listener
	servNotifier    chan bool
	cookieStore     *sessions.CookieStore
	router          *mux.Router
}

// BuvServerOptions is a structure for defining the parameters used when creating a new
// BuvServer.
type ServerOptions struct {
	// FileLog is the root name of the file to log to. A timestamp and file suffix
	// will be applied to the name.
	FileLog string

	// DirectoryLog is the path where logging files will be placed and must be terminated
	// by the directory separator character ('/' for unix-based systems, '\' for others)
	DirectoryLog string

	// FilePermissions specifies the logging file permissions when new files are created.
	// It is suggested to set up proper permissions and ownerships and use a different
	// value than the very permissible 0666, such as 0644, to prevent abuse.
	FilePermissions os.FileMode

	// DirectoryPermissions specifies the logging directory permissions if the path is
	// created and does not already exist. It is suggested to set up proper permissions
	// and ownerships if necessary to prevent abuse.
	DirectoryPermissions os.FileMode

	// AuthenticationKeySize determines the strength of the authentication key used in the session
	// cookie store. It must be 32 or 64. Only used if the GenerateKeys field is true.
	AuthenticationKeySize int

	// EncryptionKeySize determines the strength of the encryption key used in the session cookie
	// store. It must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256 modes. Only used if
	// the GenerateKeys field is true.
	EncryptionKeySize int

	// The path of the cookie -- determines which paths in the domain to send the cookies
	// along with. "/" would specify for all paths in the host domain.
	CookiePath string

	// The maximum age of the cookie before expiration, in seconds.
	MaxAge int

	// Whether the cookie is modifiable only through HTTP requests (recommended value: true).
	HttpOnly bool

	// Whether to use the AuthenticationKeySize & EncryptionKeySize fields in the ServerOptions
	// to automatically generate new keys. If false, uses the KeyPairs field for the cookie store.
	GenerateKeys bool

	// Alternating Authentication and Encryption keys to use if they are not being generated for
	// the cookie store. Only used if the GenerateKeys field is false.
	KeyPairs [][]byte

	// The name of the config file to save these options to, if specified, so a server can be
	// constructed using NewServerFromConfig. A value of "" will not save a copy of these options.
	ConfigFile string

	// The path to the directory containing the template files
	TemplatePath string

	// The extension used by the template files
	TemplateExtension string
}

// NewServerFromConfig creates a new Server from a JSON file representing a ServerOptions
// struct. It returns a non-nil error if a failure occurs.
func NewServerFromConfig(configPath string) (w *Server, e error) {
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var opts ServerOptions
	err = yaml.Unmarshal(bytes, &opts)
	if err != nil {
		return nil, err
	}
	return NewServer(&opts)
}

// NewServer creates a new web Server from the specified options. It returns a non-nil error
// if a failure in creation occurs.
func NewServer(options *ServerOptions) (w *Server, e error) {
	logger := dailyLogger.NewDailyLogger(options.FileLog, options.DirectoryLog, options.FilePermissions, options.DirectoryPermissions)
	logger.Start()
	var tempStore *sessions.CookieStore
	if options.GenerateKeys {
		options.KeyPairs = append(options.KeyPairs, []byte(securecookie.GenerateRandomKey(options.AuthenticationKeySize)))
		options.KeyPairs = append(options.KeyPairs, []byte(securecookie.GenerateRandomKey(options.EncryptionKeySize)))
	}
	tempStore = sessions.NewCookieStore(options.KeyPairs...)
	tempStore.Options = &sessions.Options{
		Path:     options.CookiePath,
		MaxAge:   options.MaxAge,
		HttpOnly: options.HttpOnly,
	}
	watcher, err := goTem.NewHTMLTemplateWatcher(options.TemplatePath, options.TemplateExtension, logger)
	if err != nil {
		return nil, err
	}

	server := Server{watcher, make(map[string]HandlerFunction), logger, nil, nil, tempStore, mux.NewRouter()}
	logger.Println("Successfully made buv.Server")
	if options.ConfigFile != "" {
		err := server.SaveConfigFile(options)
		if err != nil {
			logger.Println("Error saving config to " + options.ConfigFile + " : " + err.Error())
		} else {
			logger.Println("Successfully saved config file to: " + options.ConfigFile)
		}
	} else {
		logger.Println("Not saving configuration to file")
	}
	return &server, nil
}

func (b *Server) SaveConfigFile(options *ServerOptions) error {
	bytes, err := yaml.Marshal(options)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(options.ConfigFile, bytes, options.FilePermissions)
}

func (b *Server) Localhost(URLName string) {
	b.Domain("", URLName)
}

func (b *Server) Domain(domain, URLName string) {
	b.logger.Println("Using \"" + domain + "\" as the host.")
	b.router.Host(domain).Name(URLName)
}

func (b *Server) NotFoundHandler(noHandler HandlerFunction) {
	b.router.NotFoundHandler = b.handler(noHandler)
}

func (b *Server) GetUrl(URLName string, pathVars map[string]string) *url.URL {
	route := b.router.Get(URLName)
	if route == nil {
		b.logger.Println("GetUrl: No mux.Route with registered name: " + URLName)
		return nil
	}
	pathVarSlice := make([]string, len(pathVars)*2)
	index := 0
	for key, value := range pathVars {
		pathVarSlice[index] = key
		index++
		pathVarSlice[index] = value
		index++
	}
	url, err := route.URL(pathVarSlice...)
	if err != nil {
		b.logger.Println("GetUrl: " + err.Error())
		return nil
	} else {
		return url
	}
}

// AddHandleFunc adds a handler function to the web server.
// -schemes         The request schemes to support by the handler (eg "http", "https", or in the case of localhost, "").
// -path            The URI/URA to handle.
// -URLName         A unique name to call the URL so it can be reconstructed later if desired.
// -handleFunc      The handler function for the specified URI/URA by the path.
// -redirectors	    A list of functions that act before the handling function & act as a gateway before calling the handler. The
//                       handler is not called if one of the redirectors redirects.
// -methods         The HTTP methods to handle (eg "GET", "POST", etc).
// -queries         Optional: Any queries that must be present in order to handle. The map keys are the query keys and the map
//                       values are specific values (A value of "" matches any value).
// -URLParent       Optional: If specified, the subrouter based on the parent URI/URA is used and therefore this match will only
//                       be attempted if the parent also matches.
func (b *Server) AddHandleFunc(schemes []string, path, URLName string, handleFunc HandlerFunction, redirectors []Redirector, methods []string, queries map[string]string, URLParent string) {
	var querySlice []string = nil
	if queries != nil {
		querySlice = make([]string, len(queries)*2)
		index := 0
		for key, value := range queries {
			querySlice[index] = key
			index++
			querySlice[index] = value
			index++
		}
	}

	r := b.router
	if len(URLParent) > 0 {
		temp := b.router.Get(URLParent)
		if temp == nil {
			b.logger.Println("AddHandleFuncSubrouter parent not found: " + URLParent)
			return
		} else {
			b.logger.Println("AddHandleFunc parent found: " + URLParent)
			r = temp.Subrouter()
		}
	} else {
		b.logger.Println("AddHandleFunc no parent specified")
	}

	if len(querySlice) > 0 {
		b.logger.Println("AddHandleFunc schemes=" + strings.Join(schemes, ":") + ", URLName=" + URLName + ", path=" + path + ", methods=" + strings.Join(methods, ":") + ", queries=" + strings.Join(querySlice, ":"))
		r.HandleFunc(path, b.handler(redirectOrHandler(handleFunc, redirectors...))).Schemes(schemes...).Methods(methods...).Name(URLName).Queries(querySlice...)
	} else {
		b.logger.Println("AddHandleFunc schemes=" + strings.Join(schemes, ":") + ", URLName=" + URLName + ", path=" + path + ", methods=" + strings.Join(methods, ":") + " (no queries)")
		r.HandleFunc(path, b.handler(redirectOrHandler(handleFunc, redirectors...))).Schemes(schemes...).Methods(methods...).Name(URLName)
	}
}

// Starts up the web service, using the specified domain, template files, port address, css & javascript asset folders,
// handler map, and default handler for invalid URIs.
func (b *Server) Start(address string, assetFolderToExtension map[string]string) error {
	defer b.logger.Println(trackElapsed(time.Now(), "*Server Startup*"))
	b.logger.Start()
	b.logger.Println("Begin *Server Startup*")

	b.logger.Println("Starting up template watcher.")
	b.templateManager.Start()

	for assetFolder, assetExtension := range assetFolderToExtension {
		b.logger.Println("Adding asset handler: " + assetFolder + "{asset:[a-z0-9A-Z_]+(" + assetExtension + ")}")
		b.router.HandleFunc(""+assetFolder+"{asset:[a-z0-9A-Z_]+("+assetExtension+")}", b.assetHandler(assetFolder))
	}

	b.logger.Println("Adding favicon.ico support: /favicon.ico")
	b.router.HandleFunc("/favicon.ico", b.assetHandler(""))

	http.Handle("/", b.router)
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
func (b *Server) Shutdown() {
	defer b.logger.Println(trackElapsed(time.Now(), "*Server Shutdown*"))
	b.logger.Println("Begin *Server Shutdown*")
	b.logger.Println("Closing the listener.")
	b.listener.Close()
	b.logger.Println("Stopping the template watcher.")
	b.templateManager.Stop()
	b.logger.Println("Waiting for shutdown notification.")
	<-b.servNotifier
	b.logger.Stop()
}

func (b *Server) GetStringSessionValue(request *http.Request, sessionName string, key string) string {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return ""
	}
	val, ok := sess.Values[key]
	if !ok {
		b.logger.Println("GetStringSessionValue: no value for key=" + key)
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		b.logger.Println("GetStringSessionValue: value not type string for key=" + key)
		return ""
	}
	return strVal
}

func (b *Server) GetBoolSessionValue(request *http.Request, sessionName string, key string) bool {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return false
	}
	val, ok := sess.Values[key]
	if !ok {
		b.logger.Println("GetStringSessionValue: no value for key=" + key)
		return false
	}
	boolVal, ok := val.(bool)
	if !ok {
		b.logger.Println("GetStringSessionValue: value not type bool for key=" + key)
		return false
	}
	return boolVal
}

func (b *Server) SetSessionValue(writer http.ResponseWriter, request *http.Request, sessionName, key string, value interface{}) {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return
	}
	sess.Values[key] = value
	b.saveSession(request, writer, sess)
}

func (b *Server) HasSessionValue(request *http.Request, sessionName string, key string) bool {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return false
	}
	_, ok := sess.Values[key]
	return ok
}

func (b *Server) GetSessionValue(request *http.Request, sessionName string, key string) interface{} {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return nil
	}
	val, ok := sess.Values[key]
	if !ok {
		b.logger.Println("GetSessionValue: no value for key=" + key)
		return false
	}
	return val
}

func (b *Server) RemoveSessionValue(writer http.ResponseWriter, request *http.Request, sessionName, key string) {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return
	}
	delete(sess.Values, key)
	b.saveSession(request, writer, sess)
}

func (b *Server) HasStringSessionValue(request *http.Request, sessionName string, key string) bool {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return false
	}
	val, ok := sess.Values[key]
	if !ok {
		return false
	}
	_, ok = val.(string)
	return ok
}

func (b *Server) HasBoolSessionValue(request *http.Request, sessionName string, key string) bool {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return false
	}
	val, ok := sess.Values[key]
	if !ok {
		return false
	}
	_, ok = val.(bool)
	return ok
}

func (b *Server) SetFlashMessage(writer http.ResponseWriter, request *http.Request, sessionName, message, flashKey string) {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return
	}

	sess.AddFlash(message, flashKey)
	b.saveSession(request, writer, sess)
}

func (b *Server) GetFirstStringFlashMessage(writer http.ResponseWriter, request *http.Request, sessionName, flashKey string) string {
	messages := b.GetStringFlashMessages(writer, request, sessionName, flashKey)
	if len(messages) >= 1 {
		return messages[0]
	} else {
		return ""
	}
}

func (b *Server) GetStringFlashMessages(writer http.ResponseWriter, request *http.Request, sessionName, flashKey string) []string {
	sess := b.getSession(request, sessionName)
	if sess == nil {
		return nil
	}

	temp := sess.Flashes(flashKey)
	b.saveSession(request, writer, sess)
	strSlice := make([]string, len(temp))
	for index, obj := range temp {
		strConv, ok := obj.(string)
		if ok {
			strSlice[index] = strConv
		} else {
			b.logger.Println("GetStringFlashMessages: unsuccessful type conversion to string for index=" + strconv.Itoa(index))
		}
	}
	return strSlice
}

func (b *Server) Println(logString string) {
	b.logger.Println(logString)
}

func (b *Server) RenderTemplate(w http.ResponseWriter, tmpl string, p interface{}) {
	err := b.templateManager.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		b.logger.Println("buv.Server RenderTemplate error: " + err.Error())
	}
}
