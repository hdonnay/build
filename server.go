// Copyright 2015 Sevki <s@sevki.org>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"sevki.org/build/builder"
	"sevki.org/lib/prettyprint"
)

func server() {

	http.HandleFunc("/static/", static)
	http.HandleFunc("/graph/", graph)
	http.HandleFunc("/", index)
	log.Fatal(http.ListenAndServe(":8081", nil))

}
func index(w http.ResponseWriter, r *http.Request) {
	wd := "/Users/sevki/Code/go/src/sevki.org/build"
	f, err := os.Open(filepath.Join(wd, "graph/index.html"))
	if err != nil {
		http.Error(w, err.Error()+":\n"+filepath.Join(wd, "graph/index.html"), http.StatusNotFound)
	}
	io.Copy(w, f)
}
func static(w http.ResponseWriter, r *http.Request) {
	wd := "/Users/sevki/Code/go/src/sevki.org/build"
	f, err := os.Open(filepath.Join(wd, "graph", r.URL.Path[1:]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	io.Copy(w, f)
}
func graph(w http.ResponseWriter, r *http.Request) {
	c := builder.New()

	if c.ProjectPath == "" {
		fmt.Fprintf(os.Stderr, "You need to be in a git project.\n\n")
		printUsage()
	}
	c.Parse("//:harvey")

	count := c.Total

	go c.Execute(time.Minute*45, 0)
	for i := 0; i < count; i++ {
		select {
		case done := <-c.Done:
			doneMessage(done.GetName())

		case <-c.Error:
			continue
		}
	}

	w.Write([]byte(prettyprint.AsJSON(c.Root)))

}
