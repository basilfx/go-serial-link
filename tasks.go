package link

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (l *Link) writerTask(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Debugf("Writer task stopped.")
			return
		case message := <-l.writer:
			// Construct the line based on the message type.
			var line string

			if message.Type == MessageRequest {
				line = fmt.Sprintf("request %d %s\n", message.ID, message.Command)
			} else if message.Type == MessageResponse {
				line = fmt.Sprintf("response %d %s\n", message.ID, message.Command)
			} else if message.Type == MessageNotification {
				line = fmt.Sprintf("notify %s\n", message.Command)
			} else {
				log.Errorf("Unexpected message type: %d", message.Type)
				continue
			}

			// Send to serial device.
			log.Debugf("Link outgoing: %s", line)

			_, err := l.stream.Write([]byte(line))

			if err != nil {
				log.Errorf("Error while writing: %v", err)
				return
			}
		}
	}
}

func (l *Link) readerTask(ctx context.Context) {
	scanner := bufio.NewScanner(l.stream)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			log.Infof("Reader task stopped.")
			return
		default:
			// Pass on.
		}

		line := scanner.Text()

		if line == "" {
			continue
		}

		log.Debugf("Link incoming: %s", line)

		// Parse the line into a message.
		parts := strings.SplitN(line, " ", 2)
		message := Message{}

		if parts[0] == "request" {
			subParts := strings.SplitN(parts[1], " ", 2)
			id, err := strconv.ParseUint(subParts[0], 10, 32)

			if err != nil {
				log.Errorf("Unable to parse request identifier: %s", subParts[0])
				continue
			}

			message.Type = MessageRequest
			message.ID = uint32(id)
			message.Command = subParts[1]
		} else if parts[0] == "response" {
			subParts := strings.SplitN(parts[1], " ", 2)
			id, err := strconv.ParseUint(subParts[0], 10, 32)

			if err != nil {
				log.Errorf("Unable to parse response identifier: %s", subParts[0])
				continue
			}

			message.Type = MessageResponse
			message.ID = uint32(id)
			message.Command = subParts[1]
		} else if parts[0] == "notify" {
			message.Type = MessageNotification
			message.Command = parts[1]
		} else {
			log.Errorf("Unexpected message type: %s", parts[0])
			continue
		}

		// Notify all reading listeners of a new message.
		l.lock.RLock()

		for id, v := range l.listeners {
			select {
			case v <- message:
				continue
			default:
				log.Errorf("Channel of listener '%s' full.", id)
			}
		}

		l.lock.RUnlock()
	}

	if scanner.Err() != nil {
		log.Errorf("Error while reading: %v", scanner.Err())
		return
	}
}
