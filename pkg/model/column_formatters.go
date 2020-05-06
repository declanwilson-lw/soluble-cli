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

package model

import (
	"github.com/soluble-ai/soluble-cli/pkg/options"
	"github.com/soluble-ai/soluble-cli/pkg/util"
)

type ColumnFormatter string

const (
	TSFormat         = "ts"
	RelativeTSFormat = "relative_ts"
)

var validFormatters = []string{
	"", TSFormat, RelativeTSFormat,
}

func (t ColumnFormatter) isValid() bool {
	return util.StringSliceContains(validFormatters, string(t))
}

func (t ColumnFormatter) getFormatter(opts *options.PrintOpts) options.Formatter {
	switch t {
	case TSFormat:
		return opts.TimestampFormatter
	case RelativeTSFormat:
		return opts.RelativeTimestampFormatter
	}
	return nil
}
