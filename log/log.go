package log

import (
	log "github.com/sirupsen/logrus"
)

type Logger interface {
	Trace(message string, opts ...interface{})
	Debug(message string, opts ...interface{})
	Info(message string, opts ...interface{})
	Warning(message string, opts ...interface{})
	Error(message string, opts ...interface{})
	Fatal(message string, opts ...interface{})
	Panic(message string, opts ...interface{})
	Child(opts ...interface{}) Logger
}

type ChildLogger struct {
	l      Logger
	fields []interface{}
}

func (c *ChildLogger) Trace(message string, opts ...interface{}) {
	c.l.Trace(message, append(opts, c.fields...)...)
}

func (c *ChildLogger) Debug(message string, opts ...interface{}) {
	c.l.Debug(message, append(opts, c.fields...)...)
}

func (c *ChildLogger) Info(message string, opts ...interface{}) {
	c.l.Info(message, append(opts, c.fields...)...)
}

func (c *ChildLogger) Warning(message string, opts ...interface{}) {
	c.l.Warning(message, append(opts, c.fields...)...)
}

func (c *ChildLogger) Error(message string, opts ...interface{}) {
	c.l.Error(message, append(opts, c.fields...)...)
}

func (c *ChildLogger) Fatal(message string, opts ...interface{}) {
	c.l.Fatal(message, append(opts, c.fields...)...)
}

func (c *ChildLogger) Panic(message string, opts ...interface{}) {
	c.l.Panic(message, append(opts, c.fields...)...)
}

func (c *ChildLogger) Child(opts ...interface{}) Logger {
	return &ChildLogger{
		l:      c,
		fields: opts,
	}
}

type rootLogger struct {
}

func (r *rootLogger) Trace(message string, opts ...interface{}) {
	r.log("TRACE", message, opts)
}

func (r *rootLogger) Debug(message string, opts ...interface{}) {
	r.log("DEBUG", message, opts)
}

func (r *rootLogger) Info(message string, opts ...interface{}) {
	r.log("INFO", message, opts)
}

func (r *rootLogger) Warning(message string, opts ...interface{}) {
	r.log("WARNING", message, opts)
}

func (r *rootLogger) Error(message string, opts ...interface{}) {
	r.log("ERROR", message, opts)
}

func (r *rootLogger) Fatal(message string, opts ...interface{}) {
	r.log("FATAL", message, opts)
}

func (r *rootLogger) Panic(message string, opts ...interface{}) {
	r.log("PANIC", message, opts)
}

func (r *rootLogger) Child(opts ...interface{}) Logger {
	return &ChildLogger{
		l:      r,
		fields: opts,
	}
}

func (r *rootLogger) log(level string, message string, opts []interface{}) {
	if len(opts) > 0 && len(opts)%2 != 0 {
		panic("mismatched log key/value pairs")
	}

	var fields log.Fields
	if len(opts) > 0 {
		fields = make(log.Fields)
		for i := 0; i < len(opts); i += 2 {
			fields[opts[i].(string)] = opts[i+1]
		}
	}

	switch level {
	case "TRACE":
		log.WithFields(fields).Trace(message)
	case "DEBUG":
		log.WithFields(fields).Debug(message)
	case "INFO":
		log.WithFields(fields).Info(message)
	case "WARNING":
		log.WithFields(fields).Warning(message)
	case "ERROR":
		log.WithFields(fields).Error(message)
	case "FATAL":
		log.WithFields(fields).Fatal(message)
	case "PANIC":
		log.WithFields(fields).Panic(message)
	default:
		log.WithFields(log.Fields{"level": level}).Warning("bad log level")
		log.WithFields(fields).Info(message)
	}
}

var root = new(rootLogger)

func ModuleLogger(name string) Logger {
	return root.Child("module", name)
}
