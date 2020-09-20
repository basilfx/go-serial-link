package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/acomagu/bufpipe"
)

// DummyStream represents a dummy device.
type DummyStream struct {
	out1 io.ReadCloser
	in1  io.WriteCloser

	out2 io.ReadCloser
	in2  io.WriteCloser
}

// New returns an initialized dummy link.
func NewDummyStream() io.ReadWriteCloser {
	d := DummyStream{}

	d.out1, d.in1 = bufpipe.New(nil)
	d.out2, d.in2 = bufpipe.New(nil)

	// Start emulation.
	go d.emulate()

	return &d
}

func (d *DummyStream) emulate() {
	scanner := bufio.NewScanner(d.out2)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)

		if parts[0] == "request" {
			parts := strings.Split(parts[1], " ")

			switch parts[1] {
			case "ping":
				d.in1.Write([]byte(fmt.Sprintf("response %s pong\n", parts[0])))
			}
		}
	}
}

// Read implements the read method.
func (d *DummyStream) Read(p []byte) (n int, err error) {
	return d.out1.Read(p)
}

// Write implements the write method.
func (d *DummyStream) Write(p []byte) (n int, err error) {
	return d.in2.Write(p)
}

// Close will close the dummy link.
func (d *DummyStream) Close() error {
	d.out1.Close()
	d.in1.Close()

	d.out2.Close()
	d.in2.Close()

	return nil
}
