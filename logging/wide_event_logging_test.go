package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type wideWriter struct {
	http.ResponseWriter
	Status, Length int
}

func (wr *wideWriter) WriteHeader(status int) {
	wr.ResponseWriter.WriteHeader(status)
	wr.Status = status
}

func (wr *wideWriter) Write(b []byte) (int, error) {
	n, err := wr.ResponseWriter.Write(b)
	wr.Length += n
	if wr.Status == 0 {
		wr.Status = http.StatusOK
	}
	return n, err
}

func WideEventLog(zl *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wideWriter := &wideWriter{ResponseWriter: w}
		next.ServeHTTP(wideWriter, r)
		addr, _, _ := net.SplitHostPort(r.RemoteAddr)
		zl.Info("wide event example",
			zap.Int("status_code", wideWriter.Status),
			zap.Int("response_length", wideWriter.Length),
			zap.Int64("content_length", r.ContentLength),
			zap.String("method", r.Method),
			zap.String("proto", r.Proto),
			zap.String("remote_addr", addr),
			zap.String("uri", r.RequestURI),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}

var encoderCfg2 = zapcore.EncoderConfig{
	MessageKey: "msg",
	NameKey:    "name",

	LevelKey:    "level",
	EncodeLevel: zapcore.LowercaseLevelEncoder,

	CallerKey:    "caller",
	EncodeCaller: zapcore.FullCallerEncoder,

	// TimeKey:    "time",
	// EncodeTime: zapcore.ISO8601TimeEncoder,
}

func Test(t *testing.T) {
	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg2),
			zapcore.Lock(os.Stdout),
			zapcore.DebugLevel,
		),
	)
	defer func() { logger.Sync() }()
	server := httptest.NewServer(WideEventLog(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func(r io.ReadCloser) {
			_, _ = io.Copy(io.Discard, r)
			_ = r.Close()
		}(r.Body)
		_, _ = w.Write([]byte("hello"))
	})))
	defer server.Close()
	resp, err := http.Get(server.URL + "/test")
	if err != nil {
		t.Fatal(err)
	}

	_ = resp.Body.Close()
}
