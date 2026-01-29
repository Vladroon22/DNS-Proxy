package logger

import "log"

type LogEntry struct {
	Info string
}

type Logger struct {
	logChan chan LogEntry
	done    chan struct{}
}

func NewLogger() *Logger {
	return &Logger{
		logChan: make(chan LogEntry, 100),
		done:    make(chan struct{}),
	}
}

func (l *Logger) Log(entry LogEntry) {
	select {
	case l.logChan <- entry:
	default:
		log.Println("buffer full, dropping log entry")
	}
}

func (l *Logger) StartLogging() {
	for {
		select {
		case entry := <-l.logChan:
			log.Printf("[%s]\n", entry.Info)
		case <-l.done:
			for {
				select {
				case entry := <-l.logChan:
					log.Printf("[%s]\n", entry.Info)
				default:
					return
				}
			}
		}
	}
}

func (l *Logger) Stop() {
	close(l.done)
}
