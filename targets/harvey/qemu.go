// Copyright 2016 Sevki <s@sevki.org>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package harvey // import "sevki.org/build/targets/harvey"

import (
	"bufio"
	"bytes"
	"fmt"
	"sync"

	"sevki.org/build"
)

type Qemu struct {
	Name         string   `qemu:"name"`
	Dependencies []string `qemu:"deps"`
	CPU          string   `qemu:"cpu"`
	SMP          string   `qemu:"smp"`
	Memory       string   `qemu:"memory"`
	SDL          bool     `qemu:"sdl"`
	Serial       string   `qemu:"serial"`
	Append       string   `qemu:"append"`
	NoGraphic    bool     `qemu:"nographic"`
	Kernel       string   `qemu:"kernel"`
	Net          []string `qemu:"net"`
	Redir        []string `qemu:"redir"`
	Prompt       string   `qemu:"prompt"`
	Machine      string   `qemu:"machine"`
	Commands     []string `qemu:"cmds"`
	Monitor      string   `qemu:"monitor"`
}

func (q *Qemu) GetName() string {
	return q.Name
}

func (q *Qemu) GetDependencies() []string {
	return q.Dependencies
}

func (q *Qemu) Hash() []byte {
	return []byte{}
}

func (q *Qemu) Build(c *build.Context) error {
	system := "qemu-system-x86_64"
	params := []string{"-s"} // shorthand for -gdb tcp::1234

	if q.CPU != "" {
		params = append(params, "-cpu")
		params = append(params, q.CPU)
	}
	if q.SMP != "" {
		params = append(params, "-smp")
		params = append(params, q.SMP)
	}
	if q.Memory != "" {
		params = append(params, "-m")
		params = append(params, q.Memory)
	}
	if q.Serial != "" {
		params = append(params, "-serial")
		params = append(params, q.Serial)
	}
	if q.Machine != "" {
		params = append(params, "--machine")
		params = append(params, q.Machine)
	}
	if q.NoGraphic {
		params = append(params, "-nographic")
	}
	if q.Monitor != "" {
		params = append(params, "-monitor")
		params = append(params, q.Monitor)
	}
	if len(q.Net) > 0 {
		for _, k := range q.Net {
			params = append(params, "-net")
			params = append(params, fmt.Sprintf("%s", k))
		}
	}
	if len(q.Redir) > 0 {
		for _, k := range q.Redir {
			params = append(params, "-redir")
			params = append(params, fmt.Sprintf("%s", k))
		}
	}
	if q.Append != "" {
		params = append(params, "-append")
		params = append(params, fmt.Sprintf("%q", q.Append))
	}
	if q.Kernel != "" {
		params = append(params, "-kernel")
		params = append(params, q.Kernel)
	}
	fmt.Println(append([]string{system}, params...))
	x := c.Run(system, nil, params)
	var wg sync.WaitGroup
	stdIn, err := x.StdinPipe()
	if err != nil {
		return err
	}
	stdOut, err := x.StdoutPipe()
	if err != nil {
		return err
	}
	stdErr, err := x.StderrPipe()
	if err != nil {
		return err
	}
	wg.Add(2)

	go func() {
		scanner := bufio.NewScanner(stdErr)
		for scanner.Scan() {
			c.Println(scanner.Text())
		}
		wg.Done()
	}()

	go func() {
		scanner := bufio.NewScanner(stdOut)
		scanLines := func(data []byte, atEOF bool) (advance int, token []byte, err error) {

			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			if len(data) == len(q.Prompt) {
				if string(data) == q.Prompt {
					if len(q.Commands) == 0 {
						return len(data), dropCR(data), nil
					} else {
						cmd := q.Commands[0]
						q.Commands = q.Commands[:1]
						c.Printf("%s%s\n", q.Prompt, cmd)
						fmt.Printf("%s%s\n", q.Prompt, cmd)

						fmt.Fprintf(stdIn, "%s\r\n", cmd)

						return 0, nil, nil
					}
				}
			}
			if i := bytes.IndexByte(data, '\n'); i >= 0 {
				// We have a full newline-terminated line.
				return i + 1, dropCR(data[0:i]), nil
			}
			// If we're at EOF, we have a final, non-terminated line. Return it.
			if atEOF {
				return len(data), dropCR(data), nil
			}
			// Request more data.
			return 0, nil, nil
		}

		scanner.Split(scanLines)

		for scanner.Scan() {
			line := scanner.Text()
			c.Println(line)
			fmt.Println(line)
		}

		wg.Done()
	}()

	if err := x.Run(); err != nil {
		return err
	}

	wg.Wait()
	return nil
}

func (q *Qemu) Installs() map[string]string {
	return nil
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}