package main

import (
	link "github.com/basilfx/go-serial-link"

	log "github.com/sirupsen/logrus"
)

func main() {
	d := NewDummyStream()
	l := link.New()

	go l.Serve(d)

	// Send a ping request, wait for the response.
	r, err := l.Request(link.Message{
		Command: "ping",
	})

	if err != nil {
		log.Fatalf("This is unexpected.")
		return
	}

	log.Infof("Response: %s", r.Command)

	// Send another command that does not exist.
	r, err = l.Request(link.Message{
		Command: "error",
	})

	if err != nil {
		log.Errorf("This is expected.")
		return
	}
}
