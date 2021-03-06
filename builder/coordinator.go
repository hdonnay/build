// Copyright 2015-2016 Sevki <s@sevki.org>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builder // import "sevki.org/build/builder"

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"io/ioutil"

	"sevki.org/build/util"

	"strings"

	"sevki.org/build"
)

const (
	SCSSLOG = "success"
	FAILLOG = "fail"
	TMP     = "/tmp"
)

func (b *Builder) Execute(d time.Duration, r int) {

	for i := 0; i < r; i++ {

		go b.work(b.BuildQueue, i)
		b.Updates <- Update{
			Worker:    i,
			TimeStamp: time.Now(),
			Status:    Pending,
		}
	}

	go func() {
		if d > 0 {
			time.Sleep(d)
			b.Timeout <- true
		}
	}()
	if b.Root == nil {
		log.Fatal("Couldn't find the root node.")
	}
	b.visit(b.Root)
}

func (b *Builder) build(n *Node) (err error) {
	var buildErr error

	nodeHash := fmt.Sprintf("%s-%x", n.Target.GetName(), n.HashNode())
	outDir := filepath.Join(
		"/tmp",
		"build",
		nodeHash,
	)
	// check if this node was build before
	if _, err := os.Lstat(outDir); !os.IsNotExist(err) {
		if file, err := os.Open(filepath.Join(outDir, FAILLOG)); err == nil {
			errString, _ := ioutil.ReadAll(file)
			return fmt.Errorf("%s", errString)
		} else if _, err := os.Lstat(filepath.Join(outDir, SCSSLOG)); err == nil {
			return nil
		}
	}

	os.MkdirAll(outDir, os.ModeDir|os.ModePerm)

	// check failed builds.
	for _, e := range n.Children {
		if e.Status == Fail {
			buildErr = fmt.Errorf("dependency %s failed to build", e.Target.GetName())
		} else {

			for dst, src := range e.Target.Installs() {

				target := filepath.Base(dst)
				targetDir := strings.TrimRight(dst, target)

				if targetDir != "" {
					if err := os.MkdirAll(
						filepath.Join(
							outDir,
							targetDir,
						),
						os.ModeDir|os.ModePerm,
					); err != nil {
						log.Fatalf("installing dependency %s for %s: %s", e.Target.GetName(), n.Target.GetName(), err.Error())
					}
				}
				os.Symlink(
					filepath.Join(
						"/tmp",
						"build",
						fmt.Sprintf("%s-%x", e.Target.GetName(), e.HashNode()),
						src,
					),
					filepath.Join(
						outDir,
						targetDir,
						target),
				)

			}
		}
	}

	context := build.NewContext(outDir)
	n.Start = time.Now().UnixNano()

	buildErr = n.Target.Build(context)
	n.End = time.Now().UnixNano()

	logName := FAILLOG
	if buildErr == nil {
		logName = SCSSLOG
	}
	if logFile, err := os.Create(filepath.Join(outDir, logName)); err != nil {
		log.Fatalf("error creating log for %s: %s", n.Target.GetName(), err.Error())
	} else {
		errbytz, err := ioutil.ReadAll(context.Stdout())
		if err != nil {
			log.Fatalf("error reading log for %s: %s", n.Target.GetName(), err.Error())
		}
		n.Output = string(errbytz)
		_, err = logFile.Write(errbytz)
		if err != nil {
			log.Fatalf("error writing log for %s: %s", n.Target.GetName(), err.Error())
		}
		if buildErr != nil {
			return fmt.Errorf("%s: \n%s", buildErr, errbytz)
		}
	}

	return buildErr
}

func (b *Builder) work(jq chan *Node, workerNumber int) {

	for {
		select {
		case job := <-jq:
			if job.Status != Pending {
				continue
			}
			job.Lock()
			defer job.Unlock()

			job.Status = Building

			b.Updates <- Update{
				Worker:    workerNumber,
				TimeStamp: time.Now(),
				Target:    job.Url.String(),
				Status:    Started,
			}
			buildErr := b.build(job)

			if buildErr != nil {
				job.Status = Fail
				b.Updates <- Update{
					Worker:    workerNumber,
					TimeStamp: time.Now(),
					Target:    job.Url.String(),
					Status:    Fail,
				}

				b.Error <- buildErr

			} else {
				job.Status = Success

				b.Updates <- Update{
					Worker:    workerNumber,
					TimeStamp: time.Now(),
					Target:    job.Url.String(),
					Status:    Success,
				}
			}

			if !job.IsRoot {
				b.Done <- job
				job.once.Do(func() {
					for _, parent := range job.Parents {
						parent.wg.Done()
					}
				})
			} else {
				buildOut := filepath.Join(
					util.GetProjectPath(),
					"build_out",
				)
				os.RemoveAll(buildOut)
				if err := os.MkdirAll(
					buildOut,
					os.ModeDir|os.ModePerm,
				); err != nil {
					log.Fatalf("linking job %s failed: %s", job.Target.GetName(), err.Error())
				}

				for dst, src := range job.Target.Installs() {
					target := filepath.Base(dst)
					targetDir := strings.TrimRight(dst, target)

					buildOutTarget := filepath.Join(
						buildOut,
						targetDir,
					)
					if err := os.MkdirAll(
						buildOutTarget,
						os.ModeDir|os.ModePerm,
					); err != nil {
						log.Fatalf("linking job %s failed: %s", job.Target.GetName(), err.Error())
					}

					os.Symlink(
						filepath.Join(
							"/tmp",
							"build",
							fmt.Sprintf("%s-%x", job.Target.GetName(), job.HashNode()),
							src,
						),
						filepath.Join(
							buildOutTarget,
							target,
						),
					)
				}
				b.Done <- job
				close(b.Done)
				return
			}

		}
	}

}

type STATUS int

const (
	Success STATUS = iota
	Fail
	Pending
	Started
	Fatal
	Warning
	Building
)

func (b *Builder) visit(n *Node) {

	// This is not an airplane so let's make sure children get their masks on before the parents.
	for _, child := range n.Children {
		// Visit children first
		go b.visit(child)
	}

	n.wg.Wait()

	b.BuildQueue <- n
}
