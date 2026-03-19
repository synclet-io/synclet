package apphttp

import (
	"io/fs"
	"net/http"

	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/gorilla/mux"

	"github.com/synclet-io/synclet/front"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h Handler) Register(r *mux.Router) {
	distFS, _ := fs.Sub(front.DistFS, "dist")
	fileServer := http.FileServer(http.FS(distFS))

	// Use NotFoundHandler so the SPA only serves when no API route matched.
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Try serving the static file first; fall back to index.html for SPA routing.
		path := req.URL.Path
		if len(path) > 1 {
			if f, err := distFS.Open(path[1:]); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, req)
				return
			}
		}
		req.URL.Path = "/"
		fileServer.ServeHTTP(w, req)
	})
}

var _ pnphttpserver.MuxHandlerRegistrar = (*Handler)(nil)
