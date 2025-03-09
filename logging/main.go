package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type sustainedMultiWriter struct {
	writers []io.Writer
}

func (s *sustainedMultiWriter) Write(b []byte) (n int, err error) {
	var total int
	for _, r := range s.writers {
		n, wErr := r.Write(b)
		err = multierr.Append(wErr, err)
		total += n
	}
	return total, err
}

func SustainedMultiWriter(writers ...io.Writer) *sustainedMultiWriter {
	sw := &sustainedMultiWriter{writers: make([]io.Writer, 0, len(writers))}
	for _, w := range writers {
		if s, ok := w.(*sustainedMultiWriter); ok {
			sw.writers = append(sw.writers, s.writers...)
			continue
		}
		sw.writers = append(sw.writers, w)
	}
	return sw
}

func experimentMultiLogger() {
	lFile := new(bytes.Buffer)
	lDebug := log.New(os.Stdout, "DEBUG: ", log.Lshortfile)
	w := SustainedMultiWriter(lFile, lDebug.Writer())

	lEror := log.New(w, "ERROR: ", log.Llongfile)
	lEror.Println("What's up'")
	fmt.Println(lFile.String())
}

var encoderCfg = zapcore.EncoderConfig{
	MessageKey: "msg",
	NameKey:    "name",

	LevelKey:    "level",
	EncodeLevel: zapcore.LowercaseLevelEncoder,

	CallerKey:    "caller",
	EncodeCaller: zapcore.FullCallerEncoder,

	// TimeKey:    "time",
	// EncodeTime: zapcore.ISO8601TimeEncoder,
}

func zapJson() {
	zl := zap.New(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapcore.Lock(os.Stdout), zapcore.DebugLevel),
		zap.AddCaller(), zap.Fields(zap.String("version", runtime.Version())),
	)
	defer func() { _ = zl.Sync() }()
	l := zl.Named("^_^")
	l.Debug("im debug msg")
	l.Info("im info msg")
}

func zapNthOccurrence() {
	zap := zap.New(
		zapcore.NewSamplerWithOptions(
			zapcore.NewCore(
				zapcore.NewJSONEncoder(encoderCfg),
				zapcore.Lock(os.Stdout),
				zapcore.InfoLevel,
			), time.Second, 1, 3,
		),
	)
	defer func() { zap.Sync() }()

	for i := range 15 {
		zap.Info("Hello")
		zap.Info(fmt.Sprintf("%d", i))
	}

}

func zapDynamicLogger() {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { os.RemoveAll(tempDir) }()
	debugLevelFile := filepath.Join(tempDir, "level.debug")
	atomicLevel := zap.NewAtomicLevel()

	zl := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			atomicLevel,
		),
	)
	defer func() { _ = zl.Sync() }()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		zl.Fatal(err.Error())
	}
	if err = watcher.Add(tempDir); err != nil {
		zl.Fatal(err.Error())
	}
	ready := make(chan struct{})
	go func() {
		defer close(ready)
		originalLogLevel := atomicLevel.Level()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Println("1 not ok")
					return
				}
				if event.Name == debugLevelFile {
					switch {
					case event.Op&fsnotify.Create == fsnotify.Create:
						atomicLevel.SetLevel(zapcore.DebugLevel)
						ready <- struct{}{}
					case event.Op&fsnotify.Remove == fsnotify.Remove:
						atomicLevel.SetLevel(originalLogLevel)
						ready <- struct{}{}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("2 not ok")
					return
				}
				zl.Error(err.Error())
			}
		}
	}()
	zl.Debug("You can't see me")
	f, err := os.Create(debugLevelFile)
	if err != nil {
		zl.Error(err.Error())
	}
	err = f.Close()
	if err != nil {
		zl.Error(err.Error())
	}
	<-ready
	zl.Debug("My log level is debug")
	zl.Debug("I will go back to original log level")
	err = os.Remove(debugLevelFile)
	if err != nil {
		zl.Error(err.Error())
	}
	<-ready
	zl.Debug("you can't see me")
}

func main() {
	// zapJson()
	// zapNthOccurrence()
	zapDynamicLogger()
}
