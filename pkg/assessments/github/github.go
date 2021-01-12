package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/soluble-ai/go-jnode"
	"github.com/soluble-ai/soluble-cli/pkg/assessments"
	"github.com/soluble-ai/soluble-cli/pkg/log"
	"golang.org/x/oauth2"
)

type Integration struct {
	owner  string
	repo   string
	commit string
	gh     *github.Client
}

func NewIntegration(ctx context.Context, config *jnode.Node) assessments.PRIntegration {
	gitRepo := config.Path("gitRepo").AsText()
	if gitRepo == "" {
		return nil
	}
	p := strings.Split(gitRepo, "/")
	if len(p) == 3 && p[0] == "github.com" {
		return &Integration{
			owner:  p[1],
			repo:   p[2],
			commit: config.Path("gitCommit").AsText(),
			gh: github.NewClient(oauth2.NewClient(ctx,
				oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: config.Path("token").AsText(),
				}))),
		}
	}
	return nil
}

func (it *Integration) Update(ctx context.Context, assessments assessments.Assessments) {
	if len(assessments) == 0 {
		return
	}
	for _, a := range assessments {
		if a.URL == "" {
			continue
		}
		output := &github.CheckRunOutput{
			Title:   &a.Title,
			Summary: &a.Markdown,
			Text:    stringp(fmt.Sprintf("See <%s>", a.URL)),
		}
		checkRunOptions := github.CreateCheckRunOptions{
			HeadSHA:     it.commit,
			Name:        a.Title,
			DetailsURL:  &a.URL,
			ExternalID:  &a.ID,
			Status:      stringp("completed"),
			CompletedAt: &github.Timestamp{Time: time.Now()},
			Output:      output,
		}
		if a.Failed {
			checkRunOptions.Conclusion = stringp("failure")
		} else {
			checkRunOptions.Conclusion = stringp("success")
		}
		checkRun, _, err := it.gh.Checks.CreateCheckRun(ctx, it.owner, it.repo, checkRunOptions)
		if err != nil {
			log.Warnf("Could not create github check: {danger:%s}", err)
		} else {
			log.Infof("Updated github check run {primary:%s} id %d HeadSHA %s", a.Title, checkRun.GetID(), it.commit)
		}
		var annotations []*github.CheckRunAnnotation
		for _, f := range a.Findings {
			if f.FilePath != "" && f.Line > 0 && !f.Pass {
				annotations = append(annotations, &github.CheckRunAnnotation{
					Path:            toPath(f.FilePath),
					StartLine:       intp(f.Line),
					EndLine:         intp(f.Line),
					AnnotationLevel: toAnnotationLevel(f.Severity),
					Title:           stringp(f.GetTitle()),
					Message:         stringp(truncate(f.Description, 100)),
				})
				if len(annotations) == 50 {
					it.updateCheckRun(ctx, checkRun, annotations)
					annotations = nil
				}
			}
		}
		if len(annotations) > 0 {
			it.updateCheckRun(ctx, checkRun, annotations)
		}
	}
}

func (it *Integration) updateCheckRun(ctx context.Context, checkRun *github.CheckRun, annotations []*github.CheckRunAnnotation) {
	_, _, err := it.gh.Checks.UpdateCheckRun(ctx, it.owner, it.repo, checkRun.GetID(), github.UpdateCheckRunOptions{
		Name: checkRun.GetName(),
		Output: &github.CheckRunOutput{
			Summary:     checkRun.Output.Summary,
			Title:       checkRun.Output.Title,
			Annotations: annotations,
		},
	})
	if err != nil {
		log.Warnf("Could not add annotations to github check run: {danger:%s}", err)
	}
}

func stringp(s string) *string {
	if s != "" {
		return &s
	}
	return nil
}

func intp(i int) *int {
	return &i
}

func toPath(p string) *string {
	return stringp(strings.TrimLeft(p, "/"))
}

func toAnnotationLevel(s string) *string {
	switch strings.ToLower(s) {
	case "info":
		return stringp("notice")
	default:
		return stringp("failure")
	}
}

func truncate(s string, m int) string {
	if len(s) > m {
		return s[0:m-3] + "..."
	}
	return s
}

func init() {
	assessments.RegisterPRIntegration(NewIntegration)
}
