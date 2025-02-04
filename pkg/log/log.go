// Copyright 2020 Soluble Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
	"sync"

	"github.com/fatih/color"
	"github.com/soluble-ai/go-colorize"
)

const (
	Error = iota
	Warning
	Info
	Debug
	Trace
)

type startupMessage struct {
	level    int
	template string
	args     []interface{}
}

var (
	Level      = Info
	levelNames = map[int]string{
		Error:   "Error",
		Warning: " Warn",
		Info:    " Info",
		Debug:   "Debug",
		Trace:   "Trace",
	}
	configured      bool
	startupMessages []startupMessage
	lock            sync.Mutex
)

func Log(level int, template string, args ...interface{}) {
	lock.Lock()
	defer lock.Unlock()
	if !configured {
		// defer actually logging messages until logging has been
		// configured
		startupMessages = append(startupMessages, startupMessage{
			level:    level,
			template: template,
			args:     args,
		})
		return
	}
	if level <= Level {
		colorize.Colorize("{secondary:[%s]} ", levelNames[level])
		colorize.Colorize(template, args...)
		if template[len(template)-1] != '\n' {
			fmt.Fprintln(color.Output)
		}
	}
}

func logStartupMessages() {
	configured = true
	for _, m := range startupMessages {
		Log(m.level, m.template, m.args...)
	}
	startupMessages = nil
}

func Infof(template string, args ...interface{}) {
	Log(Info, template, args...)
}

func Debugf(template string, args ...interface{}) {
	Log(Debug, template, args...)
}

func Errorf(template string, args ...interface{}) {
	Log(Error, template, args...)
}

func Warnf(template string, args ...interface{}) {
	Log(Warning, template, args...)
}

func Tracef(template string, args ...interface{}) {
	Log(Trace, template, args...)
}

type TempLevel struct {
	orig int
}

func SetTempLevel(level int) *TempLevel {
	t := &TempLevel{orig: Level}
	Level = level
	return t
}

func (l *TempLevel) Restore() {
	Level = l.orig
}
