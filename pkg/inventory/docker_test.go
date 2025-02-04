// Copyright 2021 Soluble Inc
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

package inventory

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocker(t *testing.T) {
	assert := assert.New(t)
	m := &Manifest{}
	m.scan("testdata", dockerDetector(0))
	assert.ElementsMatch(m.DockerDirectories.Values(),
		[]string{filepath.FromSlash("d/dot"), filepath.FromSlash("d/simple"), filepath.FromSlash("d/rdot")})
	assert.ElementsMatch(m.Dockerfiles.Values(),
		[]string{filepath.FromSlash("d/dot/Dockerfile.dot"),
			filepath.FromSlash("d/rdot/dot.Dockerfile"),
			filepath.FromSlash("d/simple/Dockerfile")})
}
