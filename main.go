package main

import (
	"bufio"
	"container/list"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Commit struct {
	ChangeId string

	CommitId string

	Title string
}

const kCommit = "commit "
const kMessagePrefix = "    "

// Gerrit Change-Id: prefix in commit message
const kChangeId = kMessagePrefix + "Change-Id: "

const filenameGerritCherryIgnore = ".gerrit-cherry-ignore"

func GerritCommits(commitName string) (map[string]*Commit, *list.List, error) {
	// change-Id -> commit object
	var m map[string]*Commit = make(map[string]*Commit)
	// ordered changeId
	var l *list.List = list.New()

	cmd := exec.Command("git", "log", "--decorate=short", commitName)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	var commit *Commit
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, kCommit) {
			hash := strings.TrimPrefix(line, kCommit)
			if idx := strings.IndexByte(hash, ' '); idx >= 0 {
				hash = hash[:idx]
			}
			commit = &Commit{CommitId: hash}
		} else if strings.HasPrefix(line, kChangeId) {
			commit.ChangeId = strings.TrimPrefix(line, kChangeId)
			m[commit.ChangeId] = commit
			l.PushBack(commit.ChangeId)
		} else if strings.HasPrefix(line, kMessagePrefix) {
			if len(commit.Title) == 0 {
				commit.Title = strings.TrimPrefix(line, kMessagePrefix)
			}
		}
	}

	return m, l, nil
}

func PrintCommit(c *Commit) {
	fmt.Printf("%v %v %v\n", c.ChangeId, c.CommitId, c.Title)
}

func main() {
	commit := os.Args[1]

	// read HEAD
	mapHEAD, _, err := GerritCommits("HEAD")
	if err != nil {
		log.Fatal(err)
	}

	// read other
	mapOther, listOther, err := GerritCommits(commit)
	if err != nil {
		log.Fatal(err)
	}

	// open .gerrit-cherry-ignore for explicit skipped Change-Ids
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dir, err := filepath.Abs(pwd)
	for {
		filename := dir + string(os.PathSeparator) + filenameGerritCherryIgnore
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			file, err := os.Open(filename)
			if err == nil {
				defer file.Close()
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if len(line) == 0 { // empty line
						continue
					} else if strings.HasPrefix(line, "#") { // comment
						continue
					}

					commit := &Commit{}

					// do not care about Commit.Title
					n, err := fmt.Sscanf(line, "%41s %40s", &commit.ChangeId, &commit.CommitId)
					if err == nil && n == 2 {
						mapHEAD[commit.ChangeId] = commit
					}
				}
			}
			break
		}
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// file ".gerrit-cherry-ignore" not found in system root directory
			break
		}
		dir = parentDir
	}

	// remove commits that already in HEAD
	for k, _ := range mapHEAD {
		delete(mapOther, k)
	}

	// print left commits in other branch
	for e := listOther.Front(); e != nil; e = e.Next() {
		var changeId string = e.Value.(string)
		if c := mapOther[changeId]; c != nil {
			PrintCommit(c)
		}
	}
}
