package glog

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	// DebugMode - "debug"
	DebugMode = iota
	// ProductionMode - "roduction"
	ProductionMode = iota
)

type Config struct {
	mode    int
	info    *log.Logger
	request *log.Logger
	warn    *log.Logger
	err     *log.Logger
	trace   *log.Logger
}

var config Config

func init() {
	SetMode(DebugMode)
}

// SetMode - Устанавливает режим логирования
func SetMode(mod int) {
	switch mod {

	case DebugMode:
		config.mode = DebugMode

		config.info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
		config.request = log.New(os.Stdout, "REQUEST: ", 0)
		config.warn = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime)
		config.err = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime)
		config.trace = log.New(os.Stderr, "TRACE: ", log.Ldate|log.Ltime)

	case ProductionMode:
		config.mode = ProductionMode

		config.info = log.New(os.Stdout, "", 0)
		config.request = log.New(os.Stdout, "", 0)
		config.warn = log.New(os.Stdout, "", 0)
		config.err = log.New(os.Stderr, "", 0)
		config.trace = log.New(os.Stderr, "", 0)
	default:
		panic("Неопознанный режим логера")
	}
}

// Info - выводит сообщение по формату + "\n"
func Info(format string, v ...interface{}) {
	switch config.mode {
	case DebugMode:
		config.info.Printf("["+format+"]\n", v...)
	case ProductionMode:
		config.info.Printf(`{"loglevel":"info","date":"`+time.Now().Format(time.RFC3339)+`","message":"`+format+"\"}\n", v...)
	}
}

// Warn - выводит сообщение по формату + "\n"
func Warn(format string, v ...interface{}) {
	switch config.mode {
	case DebugMode:
		config.warn.Printf("["+format+"]\n", v...)
	case ProductionMode:
		config.warn.Printf(`{"loglevel":"warn","date":"`+time.Now().Format(time.RFC3339)+`","message":"`+format+"\"}\n", v...)
	}
}

// Err - выводит сообщение по формату + "\n"
func Err(format string, v ...interface{}) {
	switch config.mode {
	case DebugMode:
		config.err.Printf("["+format+"]\n", v...)
	case ProductionMode:
		config.err.Printf(`{"loglevel":"error","date":"`+time.Now().Format(time.RFC3339)+`","message":"`+format+"\"}\n", v...)
	}
}

// Trace - выводит сообщение по формату + "\n"
func Trace(format string, v ...interface{}) {
	// DebugColor   = "\033[0;36m%s\033[0m"

	switch config.mode {
	case DebugMode:
		v = append(v, debug.Stack())
		config.trace.Printf("["+format+"]\n%s", v...)
	case ProductionMode:
		stack := debug.Stack()
		stack = bytes.Replace(stack, []byte("\n"), []byte("\\n"), -1)
		stack = bytes.Replace(stack, []byte("\t"), []byte("\\t"), -1)
		config.trace.Printf(`{"loglevel":"trace","date":"`+time.Now().Format(time.RFC3339)+`","message":"`+format+`","stacktrace":"`+string(stack)+"\"}\n", v...)
	}
}

// Telegram - отправляет сообщение по формату + "\n"
func Telegram(format string, v ...interface{}) {
	bot, err := tgbotapi.NewBotAPI("1595064321:AAGJJt3Sve-5aohdAvmN6QKui7E6wEqTOMw")
	if err != nil {
		Err(err.Error())
	}
	// log.Printf("Authorized on account %s", bot.Self.UserName)

	msg := tgbotapi.NewMessage(239313732, fmt.Sprintf("["+format+"]\n", v...))
	bot.Send(msg)
}

// GinLoger - custom loger for gin
func GinLoger() gin.HandlerFunc {

	var patern string

	if config.mode == DebugMode {
		patern = "%v | %3d | %13v | %15s |%-7s %s\n"
	} else if config.mode == ProductionMode {
		patern = `{"logLevel":"request","date": "%v","statusCode":"%d","latency":"%v","ip":"%s","method":"%s","path":"%s"}`
	}

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Stop timer
		TimeStamp := time.Now()
		Latency := TimeStamp.Sub(start)

		ClientIP := c.ClientIP()
		Method := c.Request.Method
		StatusCode := c.Writer.Status()
		// ErrorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// BodySize := c.Writer.Size()

		if raw != "" {
			path = path + "?" + raw
		}

		Path := path

		config.request.Printf(patern,
			TimeStamp.Format("2006/01/02 - 15:04:05"),
			StatusCode,
			Latency,
			ClientIP,
			Method,
			Path)
	}
}

// var out io.Writer = os.Stderr

//

// GinRecovery - custom recovery for gin
func GinRecovery() gin.HandlerFunc {
	const reset = "\033[0m"
	var logger *log.Logger = log.New(os.Stderr, "\n\n\x1b[31m", log.LstdFlags)

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				if logger != nil {
					stack := debug.Stack()

					httpRequest, _ := httputil.DumpRequest(c.Request, false)
					headers := strings.Split(string(httpRequest), "\r\n")
					for idx, header := range headers {
						current := strings.Split(header, ":")
						if current[0] == "Authorization" {
							headers[idx] = current[0] + ": *"
						}
					}
					if brokenPipe {
						logger.Printf("%s\n%s%s", err, string(httpRequest), reset)
					} else if gin.IsDebugging() {
						logger.Printf("[Recovery] %s panic recovered:\n%s\n%s\n%s%s",
							time.Now().Format(time.RFC3339), strings.Join(headers, "\r\n"), err, stack, reset)
					} else {
						logger.Printf("[Recovery] %s panic recovered:\n%s\n%s%s",
							time.Now().Format(time.RFC3339), err, stack, reset)
					}

					// Также сигналит в телегу
					Telegram("[Recovery] %s panic recovered:\n%s\n%s%s", time.Now().Format(time.RFC3339), err, stack, reset)
				}

				// If the connection is dead, we can't write a status to it.
				if brokenPipe {
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
				} else {
					c.AbortWithStatus(http.StatusInternalServerError)
				}
			}
		}()
		c.Next()
	}
}
