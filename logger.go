package logger

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DEBUG_TAG     = "[D] "
	INFO_TAG      = "[I] "
	WARNNING_TAG  = "[W] "
	ERROR_TAG     = "[E] "
	FATAL_TAG     = "[F] "
	LOG_ERROR_TAG = "[S] "
)

const (
	DEBUG_LEVEL    = 0
	INFO_LEVEL     = 1
	WARNINIG_LEVEL = 2
	ERROR_LEVEL    = 3
	PRINT_LEVEL    = 4
	PRINTF_LEVEL   = 5
	FATAL_LEVEL    = 6
)

const (
	MOVE_DEFAULT_DAYS  = 1
	MOVE_DEFAULT_LINES = 100 * 10000
	MOVE_DEFAULT_BYTES = 100 * 1024 * 1024      // 100M
	MOVE_MAX_BYTES     = 1 * 1024 * 1024 * 1024 // 1G
)

const (
	CALL_DEPTH = 2
	LOG_SUFFIX = ".log"
)

var (
	levelTag = []string{"D", "I", "W", "E", "P", "PF", "F"}
)

type LogInterface interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warning(format string, v ...interface{})
	Error(format string, v ...interface{})
	Print(content string)
	Printf(format string, v ...interface{})
	Fatal(format string, v ...interface{})
}

type MoveMethod struct {
	MaxDay   int   // 默认 1天
	MaxLines int32 // 默认 100万行
	MaxBytes int64 // 默认 100M=100 * 1024 * 1024
}

func check(val *MoveMethod) *MoveMethod {
	if val == nil || (val.MaxDay == 0 && val.MaxBytes == 0 && val.MaxLines == 0) {
		return &MoveMethod{
			MaxDay:   MOVE_DEFAULT_DAYS,
			MaxLines: MOVE_DEFAULT_LINES,
			MaxBytes: MOVE_DEFAULT_BYTES,
		}
	}
	if val.MaxBytes > MOVE_MAX_BYTES {
		val.MaxBytes = MOVE_MAX_BYTES
	}
	return val
}

func nextSplitDayUnix(day int) int64 {
	if day < 1 {
		return 0
	}
	t := time.Now()
	t1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return t1.AddDate(0, 0, day).Unix()
}

type LogInfo struct {
	level   int32
	content string
}

// 串型操作，无需加锁
// 按照规则切割文件，切割后文件命名规则为: file.2008-10-13.000.log
type Logger struct {
	osFile     *os.File      // 句柄
	logFile    string        // 输出文件名
	logPath    string        // 输出目录
	level      int32         // 默认DEBUG
	console    bool          // 默认不打印到console
	method     *MoveMethod   // 日志切割策略
	logChan    chan *LogInfo // 日志队列
	fileExpire int64         // 切割文件时间
	lines      int32         // 日志行数
	bytes      int64         // 日志大小
	fileNum    int32         // 日志切割计数
}

// 2006-01-02 03:04:05 [TAG] [source.c] log_content
func NewLogger(logPath string) *Logger {
	return NewLoggerArgs(logPath, 0, nil)
}

func NewLoggerArgs(logFullPath string, level int32, method *MoveMethod) *Logger {

	logFile, logPath := path.Base(logFullPath), path.Dir(logFullPath)

	file, err := os.OpenFile(logFullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		fmt.Printf(LOG_ERROR_TAG+"open file error! logPath:%v error:%v\n", logFullPath, err)
	}

	logger := &Logger{
		osFile:  file,
		logFile: logFile,
		logPath: logPath,
		level:   level,
		method:  check(method),
		logChan: make(chan *LogInfo, 2048),
	}

	logger.fileExpire = nextSplitDayUnix(logger.method.MaxDay)

	// reload bytes lines
	if info, err := file.Stat(); err == nil {
		logger.bytes = info.Size()
		fmt.Printf("bytes:%v\n", logger.bytes)
	}

	// reload fileNum
	if files, err := ioutil.ReadDir(logPath); err == nil {
		logger.fileNum = logger.getFileNum(files)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.serve(&wg)
	wg.Wait()

	return logger
}

func (this *Logger) SetConsole(val bool) {
	this.console = val
}

func (this *Logger) Debug(format string, v ...interface{}) {
	this.enqueue(DEBUG_LEVEL, fmt.Sprintf(format, v...))
}

func (this *Logger) Info(format string, v ...interface{}) {
	this.enqueue(INFO_LEVEL, fmt.Sprintf(format, v...))
}

func (this *Logger) Warning(format string, v ...interface{}) {
	this.enqueue(WARNINIG_LEVEL, fmt.Sprintf(format, v...))
}

func (this *Logger) Error(format string, v ...interface{}) {
	this.enqueue(ERROR_LEVEL, fmt.Sprintf(format, v...))
}

func (this *Logger) Fatal(format string, v ...interface{}) {
	this.enqueue(FATAL_LEVEL, fmt.Sprintf(format, v...))
}

func (this *Logger) Printf(format string, v ...interface{}) {
	this.enqueue(PRINTF_LEVEL, fmt.Sprintf(format, v...))
}

func (this *Logger) Print(content string) {
	this.enqueue(PRINT_LEVEL, content)
}

func (this *Logger) serve(wg *sync.WaitGroup) {
	wg.Done()
	for {
		select {
		case info := <-this.logChan:
			this.sprintf(info.level, info.content)
		}
	}
}

// [2008/01/02 08:08:08] [I] [main.go:188] content! hello world!
func (this *Logger) enqueue(level int32, content string) {
	if level == PRINT_LEVEL || level == PRINTF_LEVEL {
		this.logChan <- &LogInfo{level: level, content: content}
		return
	}

	tag := levelTag[level]
	date := time.Now().Format("2006/01/02 15:04:05")

	_, file, line, ok := runtime.Caller(CALL_DEPTH)
	if !ok {
		file, line = "???", 0
	} else {
		file = path.Base(file)
	}

	//fullContent := fmt.Sprintf("[%s] [%s] [%s:%d] %s", date, tag, file, line, content)
	fullContent := "[" + date + "] [" + tag + "] [" + file + ":" + strconv.Itoa(line) + "] " + content + "\n"

	this.logChan <- &LogInfo{level: level, content: fullContent}
}

func (this *Logger) sprintf(level int32, content string) {
	if ok := this.needMove(); ok {
		this.move()
	}
	if level < this.level {
		return
	}

	if this.console {
		fmt.Print(content)
	}

	if this.osFile != nil {
		buff := []byte(content)
		this.lines = this.lines + 1
		this.bytes = this.bytes + int64(len(buff))
		this.osFile.Write(buff)
	} else {
		if ok := this.console; !ok {
			fmt.Print(content)
		}
	}

	if level == FATAL_LEVEL {
		os.Exit(1)
	}
}

func (this *Logger) getFileNum(files []os.FileInfo) int32 {
	if files == nil || len(files) == 0 {
		return 0
	}

	core := this.logFile
	if ok := strings.Contains(this.logFile, LOG_SUFFIX); ok {
		last := len(this.logFile) - len(LOG_SUFFIX)
		core = this.logFile[:last]
	}
	date := time.Now().Format("2006-01-02")
	prefix := core + "." + date + "."

	var max int
	for _, file := range files {
		// not ignore judge dir
		name := file.Name()
		if ok := strings.Contains(name, prefix); ok {
			target := name[len(prefix):]
			if ok := strings.Contains(target, LOG_SUFFIX); ok {
				last := len(target) - len(LOG_SUFFIX)
				target = target[:last]
				if num, _ := strconv.Atoi(target); num > max {
					max = num
				}
			}
		}
	}
	return int32(max)
}

// 文件命名规则为: file.2008-10-13.000.log
func (this *Logger) genFileName() string {
	var count int
	for num := this.fileNum; num > 0; num = num / 10 {
		count++
	}

	var suffix string
	if count > 3 {
		suffix = fmt.Sprintf("%0"+strconv.Itoa(count)+"d", this.fileNum)
	} else {
		suffix = fmt.Sprintf("%03d", this.fileNum)
	}

	core := this.logFile
	if ok := strings.Contains(this.logFile, LOG_SUFFIX); ok {
		last := len(this.logFile) - len(LOG_SUFFIX)
		core = this.logFile[:last]
	}

	date := time.Now().Format("2006-01-02")
	return core + "." + date + "." + suffix + LOG_SUFFIX
}

func (this *Logger) needMove() bool {
	if (this.method.MaxLines > 0 && this.lines > this.method.MaxLines) ||
		(this.method.MaxBytes > 0 && this.bytes > this.method.MaxBytes) ||
		(this.method.MaxDay > 0 && time.Now().Unix() > this.fileExpire) {
		return true
	}
	return false
}

func (this *Logger) move() {
	this.lines = 0
	this.bytes = 0
	this.fileNum = this.fileNum + 1
	this.fileExpire = nextSplitDayUnix(this.method.MaxDay)
	oldPath := this.logPath + "/" + this.logFile
	newPath := this.logPath + "/" + this.genFileName()

	err := os.Rename(oldPath, newPath)
	if err != nil {
		fmt.Printf(LOG_ERROR_TAG+"rename log error! old:%v new:%v error:%v\n", oldPath, newPath, err)
		return
	}

	file, err := os.OpenFile(oldPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		fmt.Printf(LOG_ERROR_TAG+"open log error! path:%v error:%v\n", oldPath, err)
		return
	}

	if this.osFile != nil {
		this.osFile.Close()
	} else {
		fmt.Println(LOG_ERROR_TAG + "osfile is nil!")
	}

	this.osFile = file
}
