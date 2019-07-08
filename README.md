
```
const (
	DEBUG_LEVEL    = 0
	INFO_LEVEL     = 1
	WARNINIG_LEVEL = 2
	ERROR_LEVEL    = 3
	PRINT_LEVEL    = 4
	PRINTF_LEVEL   = 5
	FATAL_LEVEL    = 6
)
```


```
func NewLogger(logPath string) *Logger
func NewLoggerArgs(logFullPath string, level int32, method *MoveMethod) *Logger
func (this *Logger) SetConsole(val bool)
func (this *Logger) Debug(format string, v ...interface{})
func (this *Logger) Info(format string, v ...interface{})
func (this *Logger) Warning(format string, v ...interface{})
func (this *Logger) Error(format string, v ...interface{})
func (this *Logger) Fatal(format string, v ...interface{})
func (this *Logger) Printf(format string, v ...interface{})
func (this *Logger) Print(content string) 
```
