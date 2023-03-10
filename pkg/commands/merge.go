package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Xunzhuo/bitbot/cmd/prowox/config"
	"github.com/Xunzhuo/bitbot/pkg/utils"
	"github.com/tetratelabs/multierror"
	"k8s.io/klog"
)

func init() {
	registerCommand(mergeCommandName, mergeCommandFunc)
}

var mergeCommandFunc = SafeMerge
var mergeCommandName CommandName = "merge"

func SafeMerge(args ...string) error {
	if config.Get().ISSUE_KIND != "pr" {
		return errors.New("you can only merge PRs")
	}
	if HasLabel("lgtm") && HasLabel("approved") && !HasLabel("do-not-merge") {
		if err := merge(args...); err != nil {
			return err
		}
	} else {
		return errors.New("you can only merge PRs when PR has lgtm, approved label, and without do-not-merge")
	}
	return nil
}

type PRNumberList []struct {
	LatestReviews []struct {
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		AuthorAssociation   string        `json:"authorAssociation"`
		Body                string        `json:"body"`
		SubmittedAt         time.Time     `json:"submittedAt"`
		IncludesCreatedEdit bool          `json:"includesCreatedEdit"`
		ReactionGroups      []interface{} `json:"reactionGroups"`
		State               string        `json:"state"`
	} `json:"latestReviews"`
	Number int `json:"number"`
}

func (p PRNumberList) IDs() PRNumberMap {
	ids := PRNumberMap{}
	for _, pr := range p {
		if len(pr.LatestReviews) == 0 {
			ids[fmt.Sprint(pr.Number)] = "NOTEXIST"
		} else {
			if pr.LatestReviews[0].Author.Login != "github-actions" {
				ids[fmt.Sprint(pr.Number)] = "NOTGITHUBACTION"
			} else {
				ids[fmt.Sprint(pr.Number)] = pr.LatestReviews[0].State
			}
		}
	}
	return ids
}

type PRNumberMap map[string]string

func (m PRNumberMap) IDs() []string {
	ids := []string{}
	for num := range m {
		ids = append(ids, num)
	}

	return ids
}

func MergeAcceptedPRs() error {
	nums, err := ListAcceptedPRs()
	if err != nil {
		return err
	}
	var errs error
	for num, status := range nums {
		if status != "APPROVED" {
			err := approvePR(num)
			if err != nil {
				errs = multierror.Append(errs, err)
			}
		} else {
			klog.Info(num, " has been approved.")
		}
		err = mergeWithNum(num)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return err
}

func approvePR(prNum string) error {
	return utils.ExecGitHubCmd(
		"pr",
		"-R",
		config.Get().GH_REPOSITORY,
		"review",
		prNum,
		"--approve")
}

func ListAcceptedPRs() (PRNumberMap, error) {
	pending, err := ListPendingPRs()
	if err != nil {
		return nil, err
	}
	blocked, err := ListBlockedPRs()
	if err != nil {
		return nil, err
	}
	for num := range blocked {
		delete(pending, num)
	}

	klog.Info("Accepted PRs: ", pending.IDs())

	return pending, nil
}

func ListPendingPRs() (PRNumberMap, error) {
	output, err := utils.ExecGitHubCmdWithOutput("pr",
		"-R",
		config.Get().GH_REPOSITORY,
		"list",
		"--label",
		"lgtm",
		"--label",
		"approved",
		"--json",
		"number",
		"--json",
		"latestReviews",
	)
	if err != nil {
		return nil, err
	}
	nums := &PRNumberList{}
	err = json.Unmarshal([]byte(output), nums)
	if err != nil {
		return nil, err
	}

	klog.Info("Pending PRs: ", nums)

	return nums.IDs(), nil
}

func ListBlockedPRs() (PRNumberMap, error) {
	output, err := utils.ExecGitHubCmdWithOutput("pr",
		"-R",
		config.Get().GH_REPOSITORY,
		"list",
		"--label",
		"lgtm",
		"--label",
		"approved",
		"--label",
		"do-not-merge",
		"--json",
		"number",
		"--json",
		"latestReviews",
	)
	if err != nil {
		return nil, err
	}
	nums := &PRNumberList{}
	err = json.Unmarshal([]byte(output), nums)
	if err != nil {
		return nil, err
	}

	klog.Info("Blocked PRs: ", nums)

	return nums.IDs(), nil
}

func mergeWithNum(prNum string) error {
	return utils.ExecGitHubCmd(
		"pr",
		"-R",
		config.Get().GH_REPOSITORY,
		"merge",
		prNum,
		"--squash",
		"--admin")
}

func merge(args ...string) error {
	var action string
	if len(args) == 0 {
		action = "--squash"
	} else {
		action = args[0]
		if action != "rebase" && action != "squash" {
			return errors.New("unsupported merge action, only support: rebase or squash")
		}
		action = fmt.Sprintf("--%s", action)
	}
	return utils.ExecGitHubCmd(
		config.Get().ISSUE_KIND,
		"-R",
		config.Get().GH_REPOSITORY,
		"merge",
		config.Get().ISSUE_NUMBER,
		action)
}
