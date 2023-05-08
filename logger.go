package logger_lab

import (
	"compress/gzip"
	"fmt"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Level     string `yaml:"level"`
	Frequency string `yaml:"frequency"`
	Console   bool   `yaml:"console"`
	MaxSize   string `yaml:"max-size"`
	Compress  bool   `yaml:"compress"`
}

type Logger struct {
	name           string
	category       string
	path           string
	level          LogLevel
	rollFrequency  RollFrequency
	mu             sync.Mutex
	compressMu     sync.Mutex
	out            io.Writer
	file           *os.File
	maxSize        int64
	config         *Config
	fileIndex      int
	lastRotateTime time.Time
	fileWriter     *FileWriter
	logQueue       chan LogContent
}

type LogContent struct {
	Level     LogLevel
	Timestamp time.Time
	Message   string
}

type LogLevel int
type RollFrequency int

const (
	DEBUG LogLevel = iota
	INFO
	JEDI
	WARNING
	ERROR
	FATAL
)

const (
	SECONDLY RollFrequency = iota
	MINUTELY
	HOURLY
	DAILY
	WEEKLY
	MONTHLY
	YEARLY
)

var levelMapping = map[string]LogLevel{
	"debug":   DEBUG,
	"info":    INFO,
	"jedi":    JEDI,
	"warning": WARNING,
	"error":   ERROR,
	"fatal":   FATAL,
}

var rollFrequencyMapping = map[string]RollFrequency{
	"secondly": SECONDLY,
	"minutely": MINUTELY,
	"hourly":   HOURLY,
	"daily":    DAILY,
	"weekly":   WEEKLY,
	"monthly":  MONTHLY,
	"yearly":   YEARLY,
}

func (level LogLevel) toString() string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case JEDI:
		return "JEDI"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func getDateFormat(l *Logger) string {
	switch l.rollFrequency {
	case SECONDLY:
		return "2006-01-02-15-04-05"
	case MINUTELY:
		return "2006-01-02-15-04"
	case HOURLY:
		return "2006-01-02-15"
	case DAILY:
		return "2006-01-02"
	case WEEKLY:
		return "2006-01-02"
	case MONTHLY:
		return "2006-01"
	case YEARLY:
		return "2006"
	default:
		return "2006-01-02"
	}
}

type FileWriter struct {
	file *os.File
}

func getAbsolutePath(path string) string {
	// Expand ~ to the user's home directory
	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "logs"
		}
		homeDir := usr.HomeDir
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// Clean the path (resolve ., .., and //)
	cleanPath := filepath.Clean(path)

	// Get the absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "logs"
	}

	return absPath
}

func (fw *FileWriter) Stat() (os.FileInfo, error) {
	return fw.file.Stat()
}

func NewFileWriter(filename string) (*FileWriter, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileWriter{file: file}, nil
}

func (fw *FileWriter) Write(p []byte) (n int, err error) {
	return fw.file.Write(p)
}

func (fw *FileWriter) Close() error {
	return fw.file.Close()
}

func New(name, category, path, configFile string) (*Logger, error) {
	var config Config
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}
	return newLogger(name, category, path, &config), nil
}

func newLogger(name, category, path string, config *Config) *Logger {
	level, ok := levelMapping[config.Level]
	if !ok {
		level = INFO
	}

	rollFrequency, ok := rollFrequencyMapping[config.Frequency]
	if !ok {
		rollFrequency = DAILY
	}
	logger := &Logger{
		name:           name,
		category:       category,
		level:          level,
		path:           getAbsolutePath(path),
		rollFrequency:  rollFrequency,
		config:         config,
		maxSize:        getBytesFromSizeString(config.MaxSize),
		fileIndex:      1,
		lastRotateTime: time.Now(),
		logQueue:       make(chan LogContent, 1024),
	}
	logger.setOutput()

	logger.mu.Lock()
	err := compressUncompressedFilesOnStartup(logger)
	if err != nil {
		fmt.Printf("logger: %v\n", err)
	}
	logger.mu.Unlock()

	go logger.startLogging()

	return logger
}

func (l *Logger) setOutput() {
	var fileWriter io.Writer
	fileWriter, err := l.createFileWriter()
	if err != nil {
		log.Printf("logger: %v\n", err)
		fileWriter = os.Stdout
	}

	if l.config.Console {
		l.out = io.MultiWriter(os.Stdout, fileWriter)
	} else {
		l.out = fileWriter
	}
}

func (l *Logger) createFileWriter() (io.Writer, error) {
	dateFormat := getDateFormat(l)

	l.lastRotateTime = time.Now()
	logDir := filepath.Join(l.path, l.category, l.lastRotateTime.Format(dateFormat))
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		return nil, err
	}

	// Read files in the directory
	files, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}

	// Filter log files and find the highest index
	logFilePattern := regexp.MustCompile(`^(\d+)\.log(\.gz)?$`)
	maxIndex := 0
	for _, file := range files {
		if matches := logFilePattern.FindStringSubmatch(file.Name()); matches != nil {
			index, err := strconv.Atoi(matches[1])
			if err == nil && index > maxIndex {
				maxIndex = index
			}
		}
	}

	// Check if the current file has reached the maximum size
	currentFile := filepath.Join(logDir, fmt.Sprintf("%d.log", maxIndex))
	fileInfo, err := os.Stat(currentFile)
	if err == nil && fileInfo.Size() < l.maxSize {
		l.fileIndex = maxIndex
	} else {
		l.fileIndex = maxIndex + 1
	}

	// Create and open the new log file
	filename := filepath.Join(logDir, fmt.Sprintf("%d.log", l.fileIndex))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	l.file = file
	l.fileWriter = &FileWriter{file: file}
	return l.fileWriter, nil
}

func (l *Logger) rotate() {
	l.mu.Lock()
	defer l.mu.Unlock()

	dateSwitched := false

	currentDate := time.Now()

	dateFormat := getDateFormat(l)

	previousDirName := filepath.Join(l.path, l.category, l.lastRotateTime.Format(dateFormat))

	// Check if the date has changed and reset the file index if necessary
	if currentDate.Format(dateFormat) != l.lastRotateTime.Format(dateFormat) {
		l.fileIndex = 1
		dateSwitched = true
	} else {
		l.fileIndex++
	}

	// Only close the fileWriter if the date has changed, or it's a new log file
	if l.fileWriter != nil && (currentDate.Format(dateFormat) != l.lastRotateTime.Format(dateFormat) || l.fileIndex > 1) {
		_ = l.fileWriter.Close()
	}

	l.lastRotateTime = currentDate

	dirName := filepath.Join(l.path, l.category, l.lastRotateTime.Format(dateFormat))

	filename := filepath.Join(dirName, fmt.Sprintf("%d.log", l.fileIndex))
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		log.Printf("logger: %v\n", err)
		return
	}
	fileWriter, err := NewFileWriter(filename)
	if err != nil {
		log.Printf("logger: %v\n", err)
		return
	}

	l.fileWriter = fileWriter
	if l.config.Console {
		l.out = io.MultiWriter(os.Stdout, fileWriter)
	} else {
		l.out = fileWriter
	}

	// Update the reference to the current log file
	l.file = l.fileWriter.file

	if dateSwitched {
		// Compress all uncompressed files in the previous folder
		err := compressPreviousUncompressedFiles(previousDirName)
		if err != nil {
			log.Printf("logger: %v\n", err)
		}
	}
}

func (l *Logger) compress() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.config.Compress {
		if l.fileWriter != nil {
			_ = l.fileWriter.Close()
		}

		dateFormat := getDateFormat(l)

		previousFileIndex := l.fileIndex - 1
		if previousFileIndex > 0 {
			previousFilename := filepath.Join(l.path, l.category, l.lastRotateTime.Format(dateFormat), fmt.Sprintf("%d.log", previousFileIndex))
			compressedFilename := previousFilename + ".gz"

			input, err := os.Open(previousFilename)
			if err != nil {
				log.Printf("logger: %v\n", err)
				return
			}

			output, err := os.Create(compressedFilename)
			if err != nil {
				log.Printf("logger: %v\n", err)
				err := input.Close()
				if err != nil {
					log.Printf("logger: %v\n", err)
				}
				return
			}

			gw, err := gzip.NewWriterLevel(output, gzip.BestCompression)
			if err != nil {
				log.Printf("logger: %v\n", err)
			}

			_, err = io.Copy(gw, input)
			if err != nil {
				log.Printf("logger: %v\n", err)
			}

			// Close the input, output, and gzip.Writer before removing the file
			err = input.Close()
			if err != nil {
				log.Printf("logger: %v\n", err)
			}
			err = gw.Close()
			if err != nil {
				log.Printf("logger: %v\n", err)
			}
			err = output.Close()
			if err != nil {
				log.Printf("logger: %v\n", err)
			}

			err = os.Remove(previousFilename)
			if err != nil {
				log.Printf("logger: %v\n", err)
			}
		}

		filename := filepath.Join(l.path, l.category, l.lastRotateTime.Format(dateFormat), fmt.Sprintf("%d.log", l.fileIndex))
		fileWriter, err := NewFileWriter(filename)
		if err != nil {
			log.Printf("logger: %v\n", err)
			return
		}
		l.fileWriter = fileWriter
		if l.config.Console {
			l.out = io.MultiWriter(os.Stdout, fileWriter)
		} else {
			l.out = fileWriter
		}
	}
}

func compressPreviousUncompressedFiles(previousLogDir string) error {
	files, err := os.ReadDir(previousLogDir)
	if err != nil {
		return err
	}

	uncompressedLogFilePattern := regexp.MustCompile(`^(\d+)\.log$`)
	for _, file := range files {
		if matches := uncompressedLogFilePattern.FindStringSubmatch(file.Name()); matches != nil {
			inputPath := filepath.Join(previousLogDir, file.Name())
			outputPath := inputPath + ".gz"

			err = compressFile(inputPath, outputPath)
			if err != nil {
				return err
			}

			err = os.Remove(inputPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func compressUncompressedFilesOnStartup(l *Logger) error {
	logCategoryDir := filepath.Join(l.path, l.category)
	dirs, err := os.ReadDir(logCategoryDir)
	if err != nil {
		log.Printf("logger: failed to read log category directory: %v\n", err)
		return err
	}

	currentDir := time.Now().Format(getDateFormat(l))
	var dirNames []string
	for _, dir := range dirs {
		if dir.IsDir() && dir.Name() != currentDir {
			dirNames = append(dirNames, dir.Name())
		}
	}

	if len(dirNames) == 0 {
		return nil
	}

	sort.Strings(dirNames)
	lastFolder := dirNames[len(dirNames)-1]

	err = compressPreviousUncompressedFiles(filepath.Join(logCategoryDir, lastFolder))

	return err
}

func compressFile(inputPath, outputPath string) error {
	input, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer func(input *os.File) {
		err := input.Close()
		if err != nil {
			log.Printf("logger: %v\n", err)
		}
	}(input)

	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func(output *os.File) {
		err := output.Close()
		if err != nil {
			log.Printf("logger: %v\n", err)
		}
	}(output)

	gw := gzip.NewWriter(output)
	defer func(gw *gzip.Writer) {
		err := gw.Close()
		if err != nil {
			log.Printf("logger: %v\n", err)
		}
	}(gw)

	_, err = io.Copy(gw, input)
	if err != nil {
		return err
	}

	return nil
}

func getBytesFromSizeString(size string) int64 {
	size = strings.TrimSpace(size)
	unit := strings.ToUpper(size[len(size)-2:])

	value, err := strconv.ParseFloat(size[:len(size)-2], 64)
	if err != nil {
		log.Printf("logger: invalid size string: %s\n", size)
		return 8 * 1024 * 1024
	}

	var bytes int64
	switch unit {
	case "KB":
		bytes = int64(value * 1024)
	case "MB":
		bytes = int64(value * 1024 * 1024)
	case "GB":
		bytes = int64(value * 1024 * 1024 * 1024)
	default:
		log.Printf("logger: invalid size string: %s\n", size)
		return 8 * 1024 * 1024
	}
	return bytes
}

func setLogColor(level LogLevel) {
	switch level {
	case ERROR:
		color.Set(color.FgRed)
	case FATAL:
		color.Set(color.FgRed)
	case WARNING:
		color.Set(color.FgYellow)
	case JEDI:
		color.Set(color.FgGreen)
	case INFO:
		color.Set(color.Reset)
	case DEBUG:
		color.Set(color.FgBlue)
	default:
		color.Set(color.Reset)
	}
}

func (l *Logger) logf(level LogLevel, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	now := time.Now()
	timeFormatted := now.Format("2006-01-02T15:04:05.000Z07:00")

	message := fmt.Sprintf(format, v...)
	logLine := fmt.Sprintf("%s %-9s %s\n", timeFormatted, fmt.Sprintf("[%s]", level.toString()), message)

	logContent := LogContent{
		Level:     level,
		Timestamp: time.Now(),
		Message:   logLine,
	}

	l.logQueue <- logContent
}

func (l *Logger) startLogging() {
	for logLine := range l.logQueue {
		if l.file != nil {
			fileInfo, err := os.Stat(l.file.Name())
			if err != nil {
				log.Printf("logger (file stat): %v\n", err)
			} else {
				if fileInfo.Size() >= l.maxSize {
					l.compressMu.Lock()
					l.rotate()
					l.compress()
					l.compressMu.Unlock()
				}
			}
		}

		setLogColor(logLine.Level)
		_, err := l.out.Write([]byte(logLine.Message))
		if err != nil {
			log.Printf("logger (write): %v\n", err)
		}
		color.Unset()
	}
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.logf(DEBUG, format, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.logf(INFO, format, v...)
}

func (l *Logger) Jedif(format string, v ...interface{}) {
	l.logf(JEDI, format, v...)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.logf(WARNING, format, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logf(ERROR, format, v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logf(FATAL, format, v...)
	os.Exit(1)
}
