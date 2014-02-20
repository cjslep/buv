package buv

import (
	"net/http"
)

type HandlerData struct {
	w http.ResponseWriter
	r *http.Request
	server *Server
}

// BuvHandleFunc is the function clients must use when handling requests. It provides access to the web logger
// in addition to a template renderer function that enables the handler to display a particular template if desired.
type HandlerFunction func(data *HandlerData)

func (h *HandlerData) SetStringSessionValue(sessionName, key, value string) {
	h.server.SetStringSessionValue(h.w, h.r, sessionName, key, value)
}

func (h *HandlerData) GetStringSessionValue(sessionName, key string) string {
	return h.server.GetStringSessionValue(h.r, sessionName, key)
}

func (h *HandlerData) HasStringSessionValue(sessionName, key string) bool {
	return h.server.HasStringSessionValue(h.r, sessionName, key)
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
	return h.r.Method == "GET"
}

func (h *HandlerData) IsPostMethod() bool {
	return h.r.Method == "POST"
}

func (h *HandlerData) IsPutMethod() bool {
	return h.r.Method == "PUT"
}

func (h *HandlerData) IsConnectMethod() bool {
	return h.r.Method == "CONNECT"
}

func (h *HandlerData) IsTraceMethod() bool {
	return h.r.Method == "TRACE"
}

func (h *HandlerData) IsDeleteMethod() bool {
	return h.r.Method == "DELETE"
}

func (h *HandlerData) IsHeadMethod() bool {
	return h.r.Method == "HEAD"
}

func (h *HandlerData) IsOptionsMethod() bool {
	return h.r.Method == "OPTIONS"
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