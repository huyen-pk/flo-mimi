package http

import (
	"encoding/json"
	"fmt"
	"io/fs"
	stdhttp "net/http"
	"path"
	"strings"

	bootstrapapp "platform/internal/application/bootstrap"
	interactionapp "platform/internal/application/interactions"
	"platform/internal/domain/interactions"
)

type Handler struct {
	bootstrapService   bootstrapapp.Service
	interactionService interactionapp.Service
	assets             fs.FS
	index              []byte
	fileServer         stdhttp.Handler
}

func NewHandler(bootstrapService bootstrapapp.Service, interactionService interactionapp.Service, assets fs.FS) stdhttp.Handler {
	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		panic(fmt.Sprintf("read index.html: %v", err))
	}

	handler := &Handler{
		bootstrapService:   bootstrapService,
		interactionService: interactionService,
		assets:             assets,
		index:              index,
		fileServer:         stdhttp.FileServer(stdhttp.FS(assets)),
	}

	mux := stdhttp.NewServeMux()
	mux.HandleFunc("/health", handler.handleHealth)
	mux.HandleFunc("/api/bootstrap", handler.handleBootstrap)
	mux.HandleFunc("/api/analytics", handler.handleAnalytics)
	mux.HandleFunc("/api/interactions", handler.handleInteractions)
	mux.HandleFunc("/", handler.handleSPA)
	return mux
}

func (h *Handler) handleHealth(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodGet {
		writeJSON(writer, stdhttp.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	writeJSON(writer, stdhttp.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleBootstrap(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodGet {
		writeJSON(writer, stdhttp.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	model, err := h.bootstrapService.Load(request.Context())
	if err != nil {
		writeJSON(writer, stdhttp.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(writer, stdhttp.StatusOK, model)
}

func (h *Handler) handleInteractions(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodPost {
		writeJSON(writer, stdhttp.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	request.Body = stdhttp.MaxBytesReader(writer, request.Body, 1<<20)
	defer request.Body.Close()

	var command interactions.Command
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&command); err != nil {
		writeJSON(writer, stdhttp.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}

	response, err := h.interactionService.Record(request.Context(), command)
	if err != nil {
		writeJSON(writer, stdhttp.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(writer, stdhttp.StatusAccepted, response)
}

func (h *Handler) handleSPA(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodGet && request.Method != stdhttp.MethodHead {
		writeJSON(writer, stdhttp.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	cleanPath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
	if cleanPath == "" {
		cleanPath = "index.html"
	}

	if _, err := fs.Stat(h.assets, cleanPath); err == nil {
		h.fileServer.ServeHTTP(writer, request)
		return
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(stdhttp.StatusOK)
	if request.Method != stdhttp.MethodHead {
		_, _ = writer.Write(h.index)
	}
}

func (h *Handler) handleAnalytics(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodGet {
		writeJSON(writer, stdhttp.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	analytics, err := h.bootstrapService.LoadAnalytics(request.Context())
	if err != nil {
		writeJSON(writer, stdhttp.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(writer, stdhttp.StatusOK, analytics)
}

func writeJSON(writer stdhttp.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}