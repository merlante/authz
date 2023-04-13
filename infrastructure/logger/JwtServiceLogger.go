package logger

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

type LoggerKeys string

const (
	ActionKey       LoggerKeys = "Action"
	ActionResultKey LoggerKeys = "EventResult"
	RemoteAddrKey   LoggerKeys = "RemoteAddr"

	logEventSeparator = "$$"
)

type LogEvent struct {
	Type        string
	Description string
}

func NewLogEventFromString(eventTypeAndDescription string) (logEvent LogEvent) {
	typeAndDesc := strings.Split(eventTypeAndDescription, logEventSeparator)
	sliceLen := len(typeAndDesc)

	if sliceLen > 0 {
		logEvent.Type = typeAndDesc[0]
	}

	if sliceLen > 1 {
		logEvent.Description = typeAndDesc[1]
	}

	return logEvent
}

func NewLogEvent(eventType string, description ...string) LogEvent {
	res := LogEvent{
		Type: eventType,
	}

	if len(description) != 0 {
		res.Description = description[0]
	}

	return res
}

func (l LogEvent) ToString() string {
	if l.Description != "" {
		return fmt.Sprintf("%s%s%s", l.Type, logEventSeparator, l.Description)
	}

	return l.Type
}

type UHCLogger interface {
	V(level int32) UHCLogger
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Error(err error)
	Fatalf(format string, args ...interface{})
}

// Logger is a logger with a background context
var Logger = NewUHCLogger(context.Background())
var _ UHCLogger = &logger{}

type logger struct {
	context   context.Context
	level     int32
	accountID string
	// TODO username is unused, should we be logging it? Could be pii
	username string
	session  string
}

// NewUHCLogger creates a new logger instance with a default verbosity of 1
func NewUHCLogger(ctx context.Context) UHCLogger {
	logger := &logger{
		context:  ctx,
		level:    1,
		username: getUsernameFromClaims(ctx),
		session:  getSessionFromClaims(ctx),
	}
	return logger
}

func (l *logger) prepareLogPrefix(format string, args ...interface{}) string {
	orig := fmt.Sprintf(format, args...)
	prefix := ""

	if l.username != "" {
		prefix = strings.Join([]string{prefix, "user='", l.username, "' "}, "")
	}

	if event, ok := l.context.Value(ActionKey).(string); ok {
		prefix = strings.Join([]string{prefix, "action='", event, "' "}, "")
		if eventStatus, ok := l.context.Value(ActionResultKey).(string); ok {
			prefix = strings.Join([]string{prefix, "result='", eventStatus, "' "}, "")
		}
	}

	if remoteAddr, ok := l.context.Value(RemoteAddrKey).(string); ok {
		prefix = strings.Join([]string{prefix, "src_ip='", remoteAddr, "' "}, "")
	}

	if l.session != "" {
		prefix = strings.Join([]string{prefix, "session='", l.session, "' "}, "")
	}

	if l.accountID != "" {
		prefix = strings.Join([]string{prefix, "accountID='", l.accountID, "' "}, "")
	}

	return strings.Trim(prefix+orig, " ")
}

func (l *logger) V(level int32) UHCLogger {
	return &logger{
		context:   l.context,
		accountID: l.accountID,
		username:  l.username,
		session:   l.session,
		level:     level,
	}
}

func getSessionFromClaims(ctx context.Context) string {
	var claims jwt.MapClaims
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return ""
	}

	if token != nil && token.Claims != nil {
		claims = token.Claims.(jwt.MapClaims)
	}

	if claims["session_state"] != nil {
		// return username from ocm token
		return claims["session_state"].(string)
	}

	return ""
}

func getUsernameFromClaims(ctx context.Context) string {
	var claims jwt.MapClaims
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return ""
	}

	if token != nil && token.Claims != nil {
		claims = token.Claims.(jwt.MapClaims)
	}

	if claims["username"] != nil {
		// return username from ocm token
		return claims["username"].(string)
	} else if claims["preferred_username"] != nil {
		// return username from mas-sso token
		return claims["preferred_username"].(string)
	}

	return ""
}

func (l *logger) Infof(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.V(glog.Level(l.level)).Infof(prefixed)
}

func (l *logger) Warningf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Warningln(prefixed)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Errorln(prefixed)
}

func (l *logger) Error(err error) {
	glog.Error(err)
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Fatalln(prefixed)
}
