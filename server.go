// Package buv is a web server dedicated to being a slave to Web 2.0, serving up web pages
// and parsing templates and providing a logger to client handling code. It allows
// a client to specify gorilla-style mux's to specific handlers
package buv

import (
	"bitbucket.org/cjslep/dailyLogger"
	"bitbucket.org/cjslep/goTem"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/securecookie"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
	"strconv"
	"strings"
	"net/url"
	"encoding/json"
	"io/ioutil"
)

// A BuvServer is a http server that is able to gracefully start and terminate connections as it starts up
// and shuts down, in addition to handling template execution, logging details, and mapping muxes to client
// code handlers as configured.
type Server struct {
	templateManager *goTem.HTMLBoss
	handlers        map[string]HandlerFunction
	logger          dailyLogger.TimeLogger
	listener        net.Listener
	servNotifier    chan bool
	cookieStore     *sessions.CookieStore
	router          *mux.Router
}

// BuvServerOptions is a convenience structure for defining the parameters used when creating a new BuvServer
type ServerOptions struct {
	// FileLog is the root name of the file to log to
	FileLog string
	
	// DirectoryLog is the path where logging files will be placed
	DirectoryLog string
	
	// FilePermissions specifies the logging file permissions when new files are created
	FilePermissions os.FileMode
	
	// DirectoryPermissions specifies the logging directory permissions when the path is created
	DirectoryPermissions os.FileMode
	
	// AuthenticationKeySize determines the strength of the authentication key used in the session
	// cookie store. It must be 32 or 64.
	AuthenticationKeySize int
	
	// EncryptionKeySize determines the strength of the encryption key used in the session cookie
	// store. It must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256 modes.
	EncryptionKeySize int
	
	// The path of the cookie on the client side.
	CookiePath string
	
	// The maximum age of the cookie before expiration.
	MaxAge int
	
	// Whether the cookie is modifiable only through HTTP requests (recommended value: true).
	HttpOnly bool
	
	// Whether to use the Authentication/Encryption key size fields to automatically generate
	// or to use the KeyPairs for the cookie store
	GenerateKeys bool
	
	// Alternating Authentication and Encryption keys to use if they are not being generated for
	// the cookie store
	KeyPairs[][]byte
	
	// The name of the config file to save these options to if specified
	ConfigFile string
}

// Convenience function to allow time tracking. Best used when deferred.
func trackElapsed(start time.Time, name string) string {
	elapsed := time.Since(start)
	return fmt.Sprintf("%s took %s", name, elapsed)
}

func NewServerFromConfig(configPath string) (w *Server, e error) {
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var opts ServerOptions
	err = json.Unmarshal(bytes, &opts)
	if err != nil {
		return nil, err
	}
	return NewServer(&opts)
}

// Creates a new Buv web server, using fileLog as the name of the file for logging, the directory for
// storing the logs, and the file and directory permissions for the logger.
func NewServer(options *ServerOptions) (w *Server, e error) {
	logger, err := dailyLogger.NewBasicTimeLogger(options.FileLog, options.DirectoryLog, options.FilePermissions, options.DirectoryPermissions)
	if err != nil {
		return nil, err
	}
	var tempStore *sessions.CookieStore
	if options.GenerateKeys {
		options.KeyPairs = append(options.KeyPairs, []byte(securecookie.GenerateRandomKey(options.AuthenticationKeySize)))
		options.KeyPairs = append(options.KeyPairs, []byte(securecookie.GenerateRandomKey(options.EncryptionKeySize)))
	}
	tempStore = sessions.NewCookieStore(options.KeyPairs...)
	tempStore.Options = &sessions.Options{
    	Path: options.CookiePath,
    	MaxAge: options.MaxAge,
    	HttpOnly: options.HttpOnly,
	}
	server := Server{goTem.NewHTMLBoss(), make(map[string]HandlerFunction), logger, nil, nil, tempStore, mux.NewRouter()}
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
	bytes, err := json.Marshal(options)
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

func (b *Server) AddHttpGetHandleFunc(path, URLName string, handleFunc HandlerFunction, redirectors []Redirector) {
	b.AddHandleFunc([]string{"http", ""}, path, URLName, handleFunc, redirectors, []string{"GET"}, nil, "")
}

func (b *Server) AddHttpGetHandleFuncQueries(path, URLName string, handleFunc HandlerFunction, redirectors []Redirector, queries map[string]string) {
	b.AddHandleFunc([]string{"http", ""}, path, URLName, handleFunc, redirectors, []string{"GET"}, queries, "")
}

func (b *Server) AddHttpPostHandleFunc(path, URLName string, handleFunc HandlerFunction, redirectors []Redirector) {
	b.AddHandleFunc([]string{"http", ""}, path, URLName, handleFunc, redirectors, []string{"POST"}, nil, "")
}

func (b *Server) AddHttpGetHandleFuncSubrouter(path, URLName string, handleFunc HandlerFunction, redirectors []Redirector, URLParentName string) {
	b.AddHandleFunc([]string{"http", ""}, path, URLName, handleFunc, redirectors, []string{"GET"}, nil, URLParentName)
}

func (b *Server) AddHttpGetHandleFuncQueriesSubrouter(path, URLName string, handleFunc HandlerFunction, redirectors []Redirector, queries map[string]string, URLParentName string) {
	b.AddHandleFunc([]string{"http", ""}, path, URLName, handleFunc, redirectors, []string{"GET"}, queries, URLParentName)
}

func (b *Server) AddHttpPostHandleFuncSubrouter(path, URLName string, handleFunc HandlerFunction, redirectors []Redirector, URLParentName string) {
	b.AddHandleFunc([]string{"http", ""}, path, URLName, handleFunc, redirectors, []string{"POST"}, nil, URLParentName)
}

func (b *Server) GetUrl(URLName string, pathVars map[string]string) *url.URL {
	route := b.router.Get(URLName)
	if route == nil {
		b.logger.Println("GetUrl: No mux.Route with registered name: " + URLName)
		return nil
	}
	pathVarSlice := make([]string, len(pathVars) * 2)
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

func (b *Server) AddHandleFunc(schemes []string, path, URLName string, handleFunc HandlerFunction, redirectors []Redirector, methods []string, queries map[string]string, URLParent string) {
	querySlice := make([]string, len(queries) * 2)
	index := 0
	for key, value := range queries {
		querySlice[index] = key
		index++
		querySlice[index] = value
		index++
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

func (b *Server) AddTemplateFiles(templateFiles map[string]([]string), path string) error {
	for mainFile, dependentTemplates := range templateFiles {
		depTempPath := make([]string, len(dependentTemplates))
		for i, d := range dependentTemplates {
			depTempPath[i] = path + d
		}
		name, err := b.templateManager.AddTemplate(path + mainFile, depTempPath)
		if err != nil {
			b.logger.Println(err.Error())
			return err
		} else {
			b.logger.Println("Successfully added template with name=\"" + name + "\"")
		}
	}
	return nil
}

// Starts up the web service, using the specified domain, template files, port address, css & javascript asset folders,
// handler map, and default handler for invalid URIs.
func (b *Server) Start(address string, assetFolderToExtension map[string]string) error {
	defer b.logger.Println(trackElapsed(time.Now(), "*Server Startup*"))
	b.logger.Println("Begin *Server Startup*")
	
	for assetFolder, assetExtension := range assetFolderToExtension {
		b.logger.Println(assetExtension + " handler using folder: " + assetFolder)
		b.router.HandleFunc(assetFolder + "{asset:[a-z]+(" + assetExtension + ")}", b.assetHandler(assetFolder))
	}
	
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
	b.logger.Println("Waiting for shutdown notification.")
	<-b.servNotifier
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
	if len(messages) > 1 {
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
		b.logger.Println("Error renderTemplate: " + err.Error())
	}
}