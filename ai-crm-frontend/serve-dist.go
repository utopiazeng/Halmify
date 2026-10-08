package main

import (
	"bufio"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", envDefault("FRONTEND_ADDR", ":3001"), "frontend listen address")
	distDir := flag.String("dist", envDefault("DIST_DIR", "dist"), "compiled frontend dist directory")
	apiTarget := flag.String("api", envDefault("API_TARGET", "http://127.0.0.1:8091"), "backend API target")
	flag.Parse()

	if err := ensureDist(*distDir); err != nil {
		log.Fatalf("dist directory is not ready: %v", err)
	}

	target, err := url.Parse(*apiTarget)
	if err != nil {
		log.Fatalf("invalid api target %q: %v", *apiTarget, err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", loggingMiddleware(apiProxy(target)))
	mux.Handle("/", loggingMiddleware(spaFileServer(*distDir)))

	log.Printf("serving %s on %s, proxying /api/ to %s", *distDir, *addr, target)
	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func apiProxy(target *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("api proxy error: %s %s -> %s: %v", r.Method, r.URL.RequestURI(), target, err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}
	return proxy
}

func spaFileServer(distDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(distDir))
	indexPath := filepath.Join(distDir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fileServer.ServeHTTP(w, r)
			return
		}

		path := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if path == "." {
			path = "index.html"
		}
		fullPath := filepath.Join(distDir, path)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			serveStatic(w, r, fileServer, path)
			return
		}

		setCacheHeaders(w, "index.html")
		http.ServeFile(w, r, indexPath)
	})
}

func serveStatic(w http.ResponseWriter, r *http.Request, fileServer http.Handler, path string) {
	setCacheHeaders(w, path)
	if !shouldGzip(r, path) {
		fileServer.ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")
	w.Header().Del("Content-Length")
	gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		fileServer.ServeHTTP(w, r)
		return
	}
	defer gz.Close()
	fileServer.ServeHTTP(gzipResponseWriter{ResponseWriter: w, writer: gz}, r)
}

func setCacheHeaders(w http.ResponseWriter, path string) {
	name := filepath.Base(path)
	if name == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
		return
	}
	if strings.HasPrefix(filepath.ToSlash(path), "assets/") && hasFingerprint(name) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
}

func shouldGzip(r *http.Request, path string) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if r.Header.Get("Range") != "" {
		return false
	}
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".css", ".html", ".js", ".json", ".map", ".md", ".svg", ".txt", ".xml":
		return true
	default:
		return false
	}
}

func hasFingerprint(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 3 {
		return false
	}
	hash := parts[len(parts)-2]
	if len(hash) < 8 {
		return false
	}
	for _, char := range hash {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (w gzipResponseWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (r *statusRecorder) ReadFrom(reader io.Reader) (int64, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if readerFrom, ok := r.ResponseWriter.(io.ReaderFrom); ok {
		n, err := readerFrom.ReadFrom(reader)
		r.bytes += int(n)
		return n, err
	}
	n, err := io.Copy(r.ResponseWriter, reader)
	r.bytes += int(n)
	return n, err
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		log.Printf("%s %s %d %dB %s from %s", r.Method, r.URL.RequestURI(), status, recorder.bytes, time.Since(startedAt), clientAddr(r))
	})
}

func clientAddr(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value == "" {
			continue
		}
		return strings.TrimSpace(strings.Split(value, ",")[0])
	}
	return r.RemoteAddr
}

func ensureDist(distDir string) error {
	info, err := os.Stat(distDir)
	if err != nil {
		return fmt.Errorf("stat %s: %w", distDir, err)
	}
	if !info.IsDir() {
		return errors.New("path is not a directory")
	}
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		return err
	}
	return nil
}

func envDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
