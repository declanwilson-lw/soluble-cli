package checkov

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/soluble-ai/soluble-cli/pkg/assessments"
	"github.com/soluble-ai/soluble-cli/pkg/policy"
	"github.com/soluble-ai/soluble-cli/pkg/policy/manager"
	"github.com/soluble-ai/soluble-cli/pkg/tools"
	"github.com/soluble-ai/soluble-cli/pkg/util"
	"gopkg.in/yaml.v3"
)

type checkovYAML string

var CheckovYAML manager.RuleType = checkovYAML("checkov")

func (checkovYAML) GetName() string {
	return "checkov"
}

func (checkovYAML) GetCode() string {
	return "ckv"
}

func (h checkovYAML) PrepareRules(rules []*policy.Rule, dst string) error {
	for _, rule := range rules {
		for _, target := range rule.Targets {
			ruleBody, err := h.readRule(rule, target)
			if err != nil {
				return err
			}
			util.GenericSet(&ruleBody, "metadata/id", rule.ID)
			util.GenericSet(&ruleBody, "metadata/name", rule.Metadata["title"])
			d, err := yaml.Marshal(ruleBody)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dst, fmt.Sprintf("%s-%s.yaml", target, rule.ID)), d, 0600); err != nil {
				return err
			}
		}
	}
	return nil
}

func (checkovYAML) readRule(rule *policy.Rule, target policy.Target) (map[string]interface{}, error) {
	d, err := os.ReadFile(filepath.Join(target.Path(rule), "rule.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var ruleBody map[string]interface{}
	if err := yaml.Unmarshal(d, &ruleBody); err != nil {
		return nil, fmt.Errorf("the YAML rule in %s/%s/rule.yaml is not legal yaml - %w", rule.Path, target, err)
	}
	return ruleBody, nil
}

func (h checkovYAML) ValidateRules(runOpts tools.RunOpts, rules []*policy.Rule) (validate manager.ValidateResult) {
	for _, rule := range rules {
		if e := h.validate(rule); e != nil {
			validate.Invalid++
			validate.AppendError(e)
		} else {
			validate.Valid++
		}
	}
	return
}

func (h checkovYAML) validate(rule *policy.Rule) error {
	var err error
	for _, target := range rule.Targets {
		if verr := validateSupportedTarget(rule, target); verr != nil {
			err = multierror.Append(err, verr)
		}
		_, terr := h.readRule(rule, target)
		if terr != nil {
			err = multierror.Append(err, terr)
		}
	}
	return err
}

func (checkovYAML) GetTestRunner(runOpts tools.RunOpts, target policy.Target) tools.Single {
	return getTestRunner(runOpts, target)
}

func (checkovYAML) FindRuleResult(findings assessments.Findings, id string) manager.PassFail {
	return findRuleResult(findings, id)
}

func init() {
	policy.RegisterRuleType(CheckovYAML)
}
