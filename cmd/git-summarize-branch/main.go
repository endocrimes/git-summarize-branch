package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/urfave/cli/v2"
)

func openGitRepository(overridePath string) (*git.Repository, error) {
	if overridePath != "" {
		return git.PlainOpen(overridePath)
	}

	return git.PlainOpenWithOptions(".git", &git.PlainOpenOptions{DetectDotGit: true})
}

func parseBranchArg(arg string) plumbing.ReferenceName {
	if strings.HasPrefix(arg, "refs/heads/") {
		return plumbing.ReferenceName(arg)
	} else {
		return plumbing.ReferenceName("refs/heads/" + arg)
	}
}

var baseReferenceSearchList = []plumbing.ReferenceName{
	plumbing.ReferenceName("refs/heads/develop"),
	plumbing.ReferenceName("refs/heads/main"),
	plumbing.ReferenceName("refs/heads/master"),
}

func findMostLikelyBaseRef(repo *git.Repository, baseBranchArg string) (*plumbing.Reference, error) {
	if baseBranchArg != "" {
		baseRefName := parseBranchArg(baseBranchArg)
		return repo.Reference(baseRefName, true)
	}

	for _, refName := range baseReferenceSearchList {
		baseRef, err := repo.Reference(refName, true)
		if err != nil || baseRef == nil {
			continue
		}

		return baseRef, nil
	}

	return nil, fmt.Errorf("could not determine base branch, try using --base-branch")
}

func main() {
	app := &cli.App{
		Name:  "git-summarize-branch",
		Usage: "Generate PR descriptions from Git Commits",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "C",
				Value: "",
				Usage: "Path to a Git Dir (if not .git)",
			},
			&cli.StringFlag{
				Name:  "base-branch",
				Value: "",
				Usage: "The base branch of your changes (default: search of develop, main, master)",
			},
		},
		Action: func(c *cli.Context) error {

			repo, err := openGitRepository(c.String("C"))
			if err != nil {
				return err
			}

			headRefName := plumbing.HEAD
			if c.NArg() > 0 {
				headRefName = parseBranchArg(c.Args().Get(0))
			}

			head, err := repo.Reference(headRefName, true)
			if err != nil {
				return err
			}

			base, err := findMostLikelyBaseRef(repo, c.String("base-branch"))
			if err != nil {
				return err
			}

			baseCommit, err := repo.CommitObject(base.Hash())
			if err != nil {
				return err
			}

			cIter, err := repo.Log(&git.LogOptions{From: head.Hash()})
			if err != nil {
				return err
			}

			fmt.Printf("## Commit Summary\n\n")

			for {
				commit, err := cIter.Next()
				if commit == nil {
					break
				}
				if err != nil {
					return err
				}

				if commit.Hash == baseCommit.Hash {
					break
				}

				fmt.Printf("- *%s* %s\n", commit.ID().String(), commit.Message)
			}

			return nil
		},

		EnableBashCompletion: true,
		HideVersion:          true,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
