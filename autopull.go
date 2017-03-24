package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	git "github.com/libgit2/git2go"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type Configuration struct {
	Cmd             []string
	GitRepo         string `json:"git_repo"`
	Branch          string `json:"git_branch"`
	Directory       string
	PeriodInSeconds int64 `json:"period_in_seconds"`
}

var configFile = flag.String("config", "conf.json", "help message for flagname")

func main() {
	flag.Parse()

	file, err := os.Open(*configFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err = decoder.Decode(&configuration)
	if err != nil {
		fmt.Println(err)
		return
	}

	CloneIfNeeded(configuration.Directory, configuration.GitRepo, configuration.Branch)

	var process **exec.Cmd
	for {
		killed := false
		setPeriodic(func() bool {
			repo, _ := git.OpenRepository(configuration.Directory)
			changed, _ := Pull(repo, configuration.Branch)
			if changed {
				pgid := (*process).Process.Pid
				if err == nil {
					syscall.Kill(pgid, 15) // note the minus sign
				}
				killed = true
				return false
			} else {
				return true
			}
		}, configuration.PeriodInSeconds)
		for _, cmd := range configuration.Cmd {
			subProcess := Run(cmd, configuration.Directory)
			process = &subProcess
			subProcess.Wait()
			if killed {
				break
			}
		}
		if !killed {
			break
		}
	}

}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func Run(cmd string, directory string) *exec.Cmd {
	argsWithProg := strings.Split(cmd, " ")
	command := argsWithProg[0]
	arguments := argsWithProg[1:len(argsWithProg)]
	subProcess := exec.Command(command, arguments...) //Just for testing, replace with your subProcess
	subProcess.Dir = directory

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		fmt.Println(err) //replace with logger, or anything you want
	}
	defer stdin.Close() // the doc says subProcess.Wait will close it, but I'm not sure, so I kept this line

	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	if err = subProcess.Start(); err != nil { //Use start, not run
		fmt.Println("An error occured: ", err) //replace with logger, or anything you want
	}

	io.WriteString(stdin, "4\n")

	return subProcess
}

func setPeriodic(f func() bool, seconds int64) {
	go func() {
		<-time.After(time.Duration(seconds * int64(time.Second)))
		repeat := f()
		if repeat {
			setPeriodic(f, seconds)
		}
	}()
}

func CloneIfNeeded(directory string, repoName string, branch string) {
	if noNeedClone, _ := exists(directory); !noNeedClone {
		cloneOptions := git.CloneOptions{CheckoutBranch: branch, Bare: false}
		repo, err := git.Clone(repoName, directory, &cloneOptions)
		if err != nil {
			fmt.Println("err:", err.Error())
		} else {
			fmt.Println("work:", repo.Workdir())
		}
	} else {
		repo, err := git.OpenRepository(directory)
		if err != nil {
			fmt.Println("err:", err.Error())
			return
		}
		_, err = Pull(repo, branch)
		if err != nil {
			fmt.Println("err:", err.Error())
			return
		}
	}
}

func Pull(repo *git.Repository, name string) (bool, error) {
	// Locate remote
	remote, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return false, err
	}

	// Fetch changes from remote
	if err := remote.Fetch([]string{}, nil, ""); err != nil {
		return false, err
	}

	// Get remote master
	remoteBranch, err := repo.References.Lookup("refs/remotes/origin/" + name)
	if err != nil {
		return false, err
	}

	remoteBranchID := remoteBranch.Target()
	// Get annotated commit
	annotatedCommit, err := repo.AnnotatedCommitFromRef(remoteBranch)
	if err != nil {
		return false, err
	}

	// Do the merge analysis
	mergeHeads := make([]*git.AnnotatedCommit, 1)
	mergeHeads[0] = annotatedCommit
	analysis, _, err := repo.MergeAnalysis(mergeHeads)
	if err != nil {
		return false, err
	}

	// Get repo head
	head, err := repo.Head()
	if err != nil {
		return false, err
	}

	if analysis&git.MergeAnalysisUpToDate != 0 {
		return false, nil
	} else if analysis&git.MergeAnalysisNormal != 0 {
		// Just merge changes
		if err := repo.Merge([]*git.AnnotatedCommit{annotatedCommit}, nil, nil); err != nil {
			return false, err
		}
		// Check for conflicts
		index, err := repo.Index()
		if err != nil {
			return false, err
		}

		if index.HasConflicts() {
			return false, errors.New("Conflicts encountered. Please resolve them.")
		}

		// Make the merge commit
		sig, err := repo.DefaultSignature()
		if err != nil {
			return false, err
		}

		// Get Write Tree
		treeId, err := index.WriteTree()
		if err != nil {
			return false, err
		}

		tree, err := repo.LookupTree(treeId)
		if err != nil {
			return false, err
		}

		localCommit, err := repo.LookupCommit(head.Target())
		if err != nil {
			return false, err
		}

		remoteCommit, err := repo.LookupCommit(remoteBranchID)
		if err != nil {
			return false, err
		}

		repo.CreateCommit("HEAD", sig, sig, "", tree, localCommit, remoteCommit)

		// Clean up
		repo.StateCleanup()
	} else if analysis&git.MergeAnalysisFastForward != 0 {
		// Fast-forward changes
		// Get remote tree
		remoteTree, err := repo.LookupTree(remoteBranchID)
		if err != nil {
			return false, err
		}

		// Checkout
		if err := repo.CheckoutTree(remoteTree, nil); err != nil {
			return false, err
		}

		branchRef, err := repo.References.Lookup("refs/heads/" + name)
		if err != nil {
			return false, err
		}

		// Point branch to the object
		branchRef.SetTarget(remoteBranchID, "")
		if _, err := head.SetTarget(remoteBranchID, ""); err != nil {
			return false, err
		}

	} else {
		return false, fmt.Errorf("Unexpected merge analysis result %d", analysis)
	}

	return true, nil
}
