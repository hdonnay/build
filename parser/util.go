// Copyright 2015-2016 Sevki <s@sevki.org>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser // import "sevki.org/build/parser"

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"sevki.org/build/token"
	"sevki.org/build/util"
)

func caller() (call string, file string, line int) {
	var caller uintptr
	caller, file, line, _ = runtime.Caller(2)
	name := strings.Split(runtime.FuncForPC(caller).Name(), ".")
	callName := name[len(name)-1]

	if len(callName) < 8 {
		return callName, file, line
	} else {
		return callName[7:], file, line
	}

}

func arrow(buf string, tok token.Token) string {
	ret := ""
	for i := 0; i < len(string(buf)); i++ {
		if i >= tok.Start && i <= tok.End {
			ret += "^"
			continue
		} else {
			ret += " "

		}
		switch i {
		case tok.Start - 1, tok.Start - 2, tok.Start - 3:
			ret += ">"
			break
		case tok.End + 1, tok.End + 2, tok.End + 3:
			ret += "<"
			break
		default:
			ret += " "
		}
	}
	return ret
}

func (p *Parser) expects(tok token.Token, expected ...token.Type) error {
	for _, t := range expected {
		if t == tok.Type {
			return nil
		}
	}
	name, _, _ := caller()
	errf := "%s:%d: While parsing %s were expecting %s but got %s."
	errf += "\n%s\n%s"
	return p.errorf(errf,
		p.Path,
		tok.Line,
		name,
		expected,
		p.curTok.Type,
		strings.Trim(p.lexer.LineBuffer(), "\n"),
		arrow(p.lexer.LineBuffer(), tok),
	)
}

//
//func ReadBuildFile(url TargetURL, wd string) (i *ast.File, err error) {
//
//	BUILDPATH := filepath.Join(url.BuildDir(wd, util.GetProjectPath()), "BUILD")
//	BUCKPATH := filepath.Join(url.BuildDir(wd, util.GetProjectPath()), "BUCK")
//
//	var FILEPATH string
//
//	if _, err := os.Stat(BUCKPATH); err == nil {
//		FILEPATH = BUCKPATH
//	} else if _, err := os.Stat(BUILDPATH); err == nil {
//		FILEPATH = BUILDPATH
//	} else {
//		return nil, err
//	}
//
//	i = &ast.File{}
//	ks, _ := os.Open(FILEPATH)
//	if err := New("BUILD", url.BuildDir(wd, util.GetProjectPath()), ks).Decode(i); err != nil {
//		return nil, err
//	}
//	return i, nil
//}

//
//func ReadFile(path string) (i *ast.File, err error) {
//	i = &ast.File{}
//
//	ks, err := os.Open(path)
//
//	if err != nil {
//		return nil, err
//	}
//	if err := New("BUILD", path, ks).Decode(i); err != nil {
//		return nil, err
//	}
//	return i, nil
//}

type TargetURL struct {
	Package string
	Target  string
}

func split(s string, c string, cutc bool) (string, string) {
	i := strings.Index(s, c)
	if i < 0 {
		return "", s
	}
	if cutc {
		return s[:i], s[i+len(c):]
	}
	return s[:i], s[i:]
}

func (u TargetURL) String() string {
	return fmt.Sprintf("//%s:%s", u.Package, u.Target)
}
func (u TargetURL) BuildDir(wd, p string) string {
	if u.Package == "" {
		return wd
	} else {
		return filepath.Join(p, u.Package)
	}
}
func NewTargetURLFromString(u string) (tu TargetURL) {

	switch {
	case u[:2] == "//":
		u = u[2:]
		break
	case u[0] == ':':
		if wd, err := os.Getwd(); err == nil {
			rel, err := filepath.Rel(util.GetProjectPath(), wd)
			if err == nil {
				tu.Package = rel
			} else {
				log.Fatal(err)
			}

		} else {
			log.Fatal(err)
		}

		break
	default:
		errorf := `'%s' is not a valid target.
a target url can only start with a '//' or a ':' for relative targets.`

		log.Fatalf(errorf, u)
	}
	tu.Package, tu.Target = split(u, ":", true)

	return
}
