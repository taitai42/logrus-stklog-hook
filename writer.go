package stklog

import (
	"fmt"
	"sync"
	"time"
)

// Global package channel accepting iEvents interface
// We accept either a log (LogMessage) or a "stack" (Stack)

var mutexBuffer = &sync.Mutex{}
var chanBuffer = make(chan iEvents)
var flusher = make(chan bool)
var running = false
var buffer = struct {
	Stacks []Stack
	Logs   []LogMessage
	mutex  sync.Mutex
}{}

// Normalized log adapted for Stklog API
type LogMessage struct {
	Level     int32                  `json:"level"`
	Extra     map[string]interface{} `json:"extra"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id"`
	Timestamp string                 `json:"timestamp"`
	Line      int                    `json:"line"`
	File      string                 `json:"file"`
}

// Bufferise logs and stacks
// Send requests every 5seconds and empty the buffer
func loop(trans iTransport) {
	ticker := time.NewTicker(1 * time.Second)
infiniteLoop:
	for {
		select {
		case toSend := <-chanBuffer:
			switch value := toSend.(type) {
			case *LogMessage:
				buffer.mutex.Lock()
				buffer.Logs = append(buffer.Logs, *toSend.(*LogMessage))
				buffer.mutex.Unlock()
			case *Stack:
				buffer.mutex.Lock()
				buffer.Stacks = append(buffer.Stacks, *toSend.(*Stack))
				buffer.mutex.Unlock()
			default:
				fmt.Printf("[STKLOG] %+v is an invalid iEvents object.\n", value)
			}
		case <-ticker.C:
			go trans.Send()
		case <-flusher:
			trans.Flush()
			// We don't close the channels, since if it writes into it before the program actually die/quit, it will panic ..
			break infiniteLoop
		}
	}
	flusher <- true
}

func cloneResetBuffers() ([]Stack, []LogMessage) {
	buffer.mutex.Lock()
	stacks := make([]Stack, len(buffer.Stacks))
	logs := make([]LogMessage, len(buffer.Logs))
	copy(stacks, buffer.Stacks)
	copy(logs, buffer.Logs)
	buffer.Stacks = nil
	buffer.Logs = nil
	buffer.mutex.Unlock()
	return stacks, logs
}
