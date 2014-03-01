package buv

import (
	"net/http"
	"net/url"
)

const (
	HTTP_METHOD_GET = "GET"
	HTTP_METHOD_POST = "POST"
	HTTP_METHOD_PUT = "PUT"
	HTTP_METHOD_CONNECT = "CONNECT"
	HTTP_METHOD_TRACE = "TRACE"
	HTTP_METHOD_DELETE = "DELETE"
	HTTP_METHOD_HEAD = "HEAD"
	HTTP_METHOD_OPTIONS = "OPTIONS"
	
	HTTP_SCHEME = "http"
	HTTPS_SCHEME = "https"
	LOCALHOST_SCHEME = ""
)

type HandlerData struct {
	w http.ResponseWriter
	r *http.Request
	server *Server
}

// HandlerFunction is the function clients must use when handling requests. It provides access to the specific
// request's HandlerData, through which handler functions can operate.
type HandlerFunction func(data *HandlerData)

// Redirector is a function clients can use to cause a request to be redirected. Upon successful redirection,
// true must be returned to prevent the default handler from being called.
type Redirector func (data *HandlerData) bool

func (h *HandlerData) SetSessionValue(sessionName, key string, value interface{}) {
	h.server.SetSessionValue(h.w, h.r, sessionName, key, value)
}

func (h *HandlerData) HasSessionValue(sessionName, key string) bool {
	return h.server.HasSessionValue(h.r, sessionName, key)
}

func (h *HandlerData) GetStringSessionValue(sessionName, key string) string {
	return h.server.GetStringSessionValue(h.r, sessionName, key)
}

func (h *HandlerData) HasStringSessionValue(sessionName, key string) bool {
	return h.server.HasStringSessionValue(h.r, sessionName, key)
}

func (h *HandlerData) GetBoolSessionValue(sessionName, key string) bool {
	return h.server.GetBoolSessionValue(h.r, sessionName, key)
}

func (h *HandlerData) HasBoolSessionValue(sessionName, key string) bool {
	return h.server.HasBoolSessionValue(h.r, sessionName, key)
}

func (h *HandlerData) RemoveSessionValue(sessionName, key string) {
	h.server.RemoveSessionValue(h.w, h.r, sessionName, key)
}

func (h *HandlerData) SetFlashMessage(sessionName, message, flashKey string) {
	h.server.SetFlashMessage(h.w, h.r, sessionName, message, flashKey)
}

func (h *HandlerData) GetFirstStringFlashMessage(sessionName, flashKey string) string {
	return h.server.GetFirstStringFlashMessage(h.w, h.r, sessionName, flashKey)
}

func (h *HandlerData) GetStringFlashMessages(sessionName, flashKey string) []string {
	return h.server.GetStringFlashMessages(h.w, h.r, sessionName, flashKey)
}

func (h *HandlerData) IsGetMethod() bool {
	return h.r.Method == HTTP_METHOD_GET
}

func (h *HandlerData) IsPostMethod() bool {
	return h.r.Method == HTTP_METHOD_POST
}

func (h *HandlerData) IsPutMethod() bool {
	return h.r.Method == HTTP_METHOD_PUT
}

func (h *HandlerData) IsConnectMethod() bool {
	return h.r.Method == HTTP_METHOD_CONNECT
}

func (h *HandlerData) IsTraceMethod() bool {
	return h.r.Method == HTTP_METHOD_TRACE
}

func (h *HandlerData) IsDeleteMethod() bool {
	return h.r.Method == HTTP_METHOD_DELETE
}

func (h *HandlerData) IsHeadMethod() bool {
	return h.r.Method == HTTP_METHOD_HEAD
}

func (h *HandlerData) IsOptionsMethod() bool {
	return h.r.Method == HTTP_METHOD_OPTIONS
}

func (h *HandlerData) Method() string {
	return h.r.Method
}

func (h *HandlerData) Redirect(newURI string, code int) {
	http.Redirect(h.w, h.r, newURI, code)
}

func (h *HandlerData) RenderTemplate(templateName string, templateData interface{}) {
	h.server.RenderTemplate(h.w, templateName, templateData)
}

func (h *HandlerData) Println(logString string) {
	h.server.Println(logString)
}

func (h *HandlerData) URL() *url.URL {
	return h.r.URL
}

func (h *HandlerData) Referrer() string {
	return h.r.Referer()
}

func (h *HandlerData) PostFormValue(key string) string {
	return h.r.PostFormValue(key)
}

func (h *HandlerData) Query() url.Values {
	return h.r.URL.Query()
}

func (h *HandlerData) String() string {
	return "Method=" + h.r.Method + " URL=" + h.r.URL.String() + " Scheme=" + h.r.URL.Scheme + " Host=" + h.r.URL.Host
}

func (h *HandlerData) GetUrl(URLName string, pathVars map[string]string) *url.URL {
	return h.server.GetUrl(URLName, pathVars)
}