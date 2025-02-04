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

package npmaudit

import (
	"github.com/soluble-ai/go-jnode"
	"github.com/soluble-ai/soluble-cli/pkg/assessments"
	"github.com/soluble-ai/soluble-cli/pkg/tools"
	"github.com/spf13/cobra"
)

type Tool struct {
	tools.DirectoryBasedToolOpts
}

var _ tools.Single = (*Tool)(nil)

func (t *Tool) Name() string {
	return "npm-audit"
}

func (t *Tool) Run() (*tools.Result, error) {
	args := []string{"audit", "--json"}
	exec, err := t.RunDocker(&tools.DockerTool{
		Name:                "npm-audit",
		Image:               "gcr.io/soluble-repo/soluble-npm:latest",
		DefaultNoDockerName: "npm",
		Directory:           t.GetDirectory(),
		Args:                args,
	})
	if err != nil {
		return nil, err
	}
	result := exec.ToResult(t.GetDirectory())
	if !exec.ExpectExitCode(0) {
		return result, nil
	}
	results, ok := exec.ParseJSON()
	if !ok {
		return result, nil
	}
	t.parseResults(result, results)
	return result, nil
}

func (t *Tool) parseResults(result *tools.Result, results *jnode.Node) *tools.Result {
	findings := assessments.Findings{}
	for _, data := range results.Path("advisories").Entries() {
		findings = append(findings, &assessments.Finding{
			Tool: map[string]string{
				"id":             data.Path("id").AsText(),
				"cwe":            data.Path("cwe").AsText(),
				"module":         data.Path("module_name").AsText(),
				"recommendation": data.Path("recommendation").AsText(),
				"severity":       data.Path("severity").AsText(),
				"title":          data.Path("title").AsText(),
			},
		})
	}
	result.Data = results
	result.Findings = findings
	return result
}

func (t *Tool) CommandTemplate() *cobra.Command {
	return &cobra.Command{
		Use:   "npm-audit",
		Short: "Run npm audit to find vulnerable dependencies of a npm application",
	}
}
