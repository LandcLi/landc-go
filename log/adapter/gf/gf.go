package gf

import (
	"context"
	"fmt"

	"github.com/LandcLi/landc-go/log/internal/logger"
)

// GFLogger 是goframe框架的日志适配器
type GFLogger struct {
	log logger.Logger
}

// NewGFLogger 创建一个新的goframe日志适配器
func NewGFLogger(log logger.Logger) *GFLogger {
	return &GFLogger{
		log: log,
	}
}

// Print 实现glog.ILogger接口的Print方法
func (g *GFLogger) Print(ctx context.Context, v ...interface{}) {
	g.log.Info(fmt.Sprint(v...))
}

// Printf 实现glog.ILogger接口的Printf方法
func (g *GFLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	g.log.Info(fmt.Sprintf(format, v...))
}

// Println 实现glog.ILogger接口的Println方法
func (g *GFLogger) Println(ctx context.Context, v ...interface{}) {
	g.log.Info(fmt.Sprintln(v...))
}

// Debug 实现glog.ILogger接口的Debug方法
func (g *GFLogger) Debug(ctx context.Context, v ...interface{}) {
	g.log.Debug(fmt.Sprint(v...))
}

// Debugf 实现glog.ILogger接口的Debugf方法
func (g *GFLogger) Debugf(ctx context.Context, format string, v ...interface{}) {
	g.log.Debugf(format, v...)
}

// Debugln 实现glog.ILogger接口的Debugln方法
func (g *GFLogger) Debugln(ctx context.Context, v ...interface{}) {
	g.log.Debug(fmt.Sprintln(v...))
}

// Info 实现glog.ILogger接口的Info方法
func (g *GFLogger) Info(ctx context.Context, v ...interface{}) {
	g.log.Info(fmt.Sprint(v...))
}

// Infof 实现glog.ILogger接口的Infof方法
func (g *GFLogger) Infof(ctx context.Context, format string, v ...interface{}) {
	g.log.Infof(format, v...)
}

// Infoln 实现glog.ILogger接口的Infoln方法
func (g *GFLogger) Infoln(ctx context.Context, v ...interface{}) {
	g.log.Info(fmt.Sprintln(v...))
}

// Notice 实现glog.ILogger接口的Notice方法
func (g *GFLogger) Notice(ctx context.Context, v ...interface{}) {
	g.log.Info(fmt.Sprint(v...))
}

// Noticef 实现glog.ILogger接口的Noticef方法
func (g *GFLogger) Noticef(ctx context.Context, format string, v ...interface{}) {
	g.log.Infof(format, v...)
}

// Noticeln 实现glog.ILogger接口的Noticeln方法
func (g *GFLogger) Noticeln(ctx context.Context, v ...interface{}) {
	g.log.Info(fmt.Sprintln(v...))
}

// Warning 实现glog.ILogger接口的Warning方法
func (g *GFLogger) Warning(ctx context.Context, v ...interface{}) {
	g.log.Warn(fmt.Sprint(v...))
}

// Warningf 实现glog.ILogger接口的Warningf方法
func (g *GFLogger) Warningf(ctx context.Context, format string, v ...interface{}) {
	g.log.Warnf(format, v...)
}

// Warningln 实现glog.ILogger接口的Warningln方法
func (g *GFLogger) Warningln(ctx context.Context, v ...interface{}) {
	g.log.Warn(fmt.Sprintln(v...))
}

// Error 实现glog.ILogger接口的Error方法
func (g *GFLogger) Error(ctx context.Context, v ...interface{}) {
	g.log.Error(fmt.Sprint(v...))
}

// Errorf 实现glog.ILogger接口的Errorf方法
func (g *GFLogger) Errorf(ctx context.Context, format string, v ...interface{}) {
	g.log.Errorf(format, v...)
}

// Errorln 实现glog.ILogger接口的Errorln方法
func (g *GFLogger) Errorln(ctx context.Context, v ...interface{}) {
	g.log.Error(fmt.Sprintln(v...))
}

// Critical 实现glog.ILogger接口的Critical方法
func (g *GFLogger) Critical(ctx context.Context, v ...interface{}) {
	g.log.Fatal(fmt.Sprint(v...))
}

// Criticalf 实现glog.ILogger接口的Criticalf方法
func (g *GFLogger) Criticalf(ctx context.Context, format string, v ...interface{}) {
	g.log.Fatalf(format, v...)
}

// Criticalln 实现glog.ILogger接口的Criticalln方法
func (g *GFLogger) Criticalln(ctx context.Context, v ...interface{}) {
	g.log.Fatal(fmt.Sprintln(v...))
}

// GetLevel 实现glog.ILogger接口的GetLevel方法
func (g *GFLogger) GetLevel() int {
	switch g.log.GetLevel() {
	case logger.DebugLevel:
		return 0 // glog.LEVEL_DEBUG
	case logger.InfoLevel:
		return 1 // glog.LEVEL_INFO
	case logger.WarnLevel:
		return 2 // glog.LEVEL_WARNING
	case logger.ErrorLevel:
		return 3 // glog.LEVEL_ERROR
	case logger.FatalLevel, logger.PanicLevel:
		return 4 // glog.LEVEL_CRITICAL
	default:
		return 1 // glog.LEVEL_INFO
	}
}

// SetLevel 实现glog.ILogger接口的SetLevel方法
func (g *GFLogger) SetLevel(level int) {
	switch level {
	case 0: // glog.LEVEL_DEBUG
		g.log.SetLevel(logger.DebugLevel)
	case 1, 2: // glog.LEVEL_INFO, glog.LEVEL_NOTICE
		g.log.SetLevel(logger.InfoLevel)
	case 3: // glog.LEVEL_WARNING
		g.log.SetLevel(logger.WarnLevel)
	case 4: // glog.LEVEL_ERROR
		g.log.SetLevel(logger.ErrorLevel)
	case 5: // glog.LEVEL_CRITICAL
		g.log.SetLevel(logger.FatalLevel)
	default:
		g.log.SetLevel(logger.InfoLevel)
	}
}

// GetStackLevel 实现glog.ILogger接口的GetStackLevel方法
func (g *GFLogger) GetStackLevel() int {
	return 0
}

// SetStackLevel 实现glog.ILogger接口的SetStackLevel方法
func (g *GFLogger) SetStackLevel(level int) {
	// 暂不实现
}

// GetAsync 实现glog.ILogger接口的GetAsync方法
func (g *GFLogger) GetAsync() bool {
	return false
}

// SetAsync 实现glog.ILogger接口的SetAsync方法
func (g *GFLogger) SetAsync(enabled bool) {
	// 暂不实现
}

// GetPrefix 实现glog.ILogger接口的GetPrefix方法
func (g *GFLogger) GetPrefix() string {
	return ""
}

// SetPrefix 实现glog.ILogger接口的SetPrefix方法
func (g *GFLogger) SetPrefix(prefix string) {
	// 暂不实现
}

// IsDebug 实现glog.ILogger接口的IsDebug方法
func (g *GFLogger) IsDebug() bool {
	return g.log.IsDebugEnabled()
}

// IsInfo 实现glog.ILogger接口的IsInfo方法
func (g *GFLogger) IsInfo() bool {
	return g.log.IsInfoEnabled()
}

// IsNotice 实现glog.ILogger接口的IsNotice方法
func (g *GFLogger) IsNotice() bool {
	return g.log.IsInfoEnabled()
}

// IsWarning 实现glog.ILogger接口的IsWarning方法
func (g *GFLogger) IsWarning() bool {
	return g.log.IsWarnEnabled()
}

// IsError 实现glog.ILogger接口的IsError方法
func (g *GFLogger) IsError() bool {
	return g.log.IsErrorEnabled()
}

// IsCritical 实现glog.ILogger接口的IsCritical方法
func (g *GFLogger) IsCritical() bool {
	return g.log.IsFatalEnabled()
}

// Flush 实现glog.ILogger接口的Flush方法
func (g *GFLogger) Flush(ctx context.Context) error {
	return g.log.Sync()
}

// Close 实现glog.ILogger接口的Close方法
func (g *GFLogger) Close(ctx context.Context) error {
	return g.log.Sync()
}
