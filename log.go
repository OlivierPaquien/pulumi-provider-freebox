package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const defaultLogFile = "pulumi-freebox-provider.log"

var (
	logFile   *os.File
	logPath   string
	logFileMu sync.Mutex
)

// logPaths returns paths to try for the log file (first successful open wins).
func logPaths() []string {
	if p := os.Getenv("FREEBOX_DEBUG_LOG"); p != "" {
		return []string{p}
	}
	var paths []string
	if runtime.GOOS != "windows" {
		paths = append(paths, filepath.Join("/tmp", defaultLogFile))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".pulumi", defaultLogFile))
	}
	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(os.TempDir(), defaultLogFile))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, defaultLogFile))
	}
	return paths
}

func writeLogLine(dest *os.File, msg string) {
	if dest == nil {
		return
	}
	if _, err := dest.WriteString(msg); err != nil {
		fmt.Fprintf(os.Stderr, "[freebox] log write error: %v\n", err)
	}
}

// freeboxLog writes to stderr and to a file. Tries /tmp, $HOME/.pulumi/, puis le cwd.
func freeboxLog(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if _, err := os.Stderr.WriteString(msg); err != nil {
		fmt.Fprintf(os.Stderr, "[freebox] stderr write error: %v\n", err)
	}
	logFileMu.Lock()
	defer logFileMu.Unlock()
	if logFile == nil {
		var errs []string
		for _, p := range logPaths() {
			dir := filepath.Dir(p)
			if dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", p, err))
					continue
				}
			}
			f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", p, err))
				continue
			}
			logFile = f
			logPath = p
			errs = nil
			writeLogLine(logFile, fmt.Sprintf("[freebox] log file: %s\n", logPath))
			break
		}
		if logFile == nil && len(errs) > 0 {
			os.Stderr.WriteString("[freebox] impossible d'ouvrir le fichier de log: " + fmt.Sprint(errs) + "\n")
		}
	}
	writeLogLine(logFile, msg)
}
