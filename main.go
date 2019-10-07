package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"container/list"

	"gopkg.in/src-d/go-git.v4"
	. "gopkg.in/src-d/go-git.v4/_examples"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// Gerrit change-Id prefix in commit message
const kChangeId = "Change-Id: "
// Limit log history to speedup 
const startTime = "2019-01-01T00:00:00Z"

func GerritCommits(r *git.Repository, hash plumbing.Hash) (map[string]*object.Commit, *list.List, error) {
	// change-Id -> commit object
	var m map[string]*object.Commit = make(map[string]*object.Commit)
	// ordered changeId
	var l *list.List = list.New() 

	var err error

	since, err := time.Parse(time.RFC3339, startTime)
	CheckIfError(err)


	cIter, err := r.Log(&git.LogOptions{From: hash})
	if err != nil {
		return nil, nil, err
	}

	err = cIter.ForEach(func(c *object.Commit) error {
		if since.After(c.Committer.When) {
			return plumbing.ErrInvalidType
		}

		scanner := bufio.NewScanner(strings.NewReader(c.Message))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, kChangeId) {
				changeId := strings.TrimPrefix(line, kChangeId)
				m[changeId] = c
				l.PushBack(changeId)
				break

			}
		}
		return nil
	})

	return m, l, nil
}

func PrintCommit(c *object.Commit, changeId string) {
	short := strings.SplitN(c.Message, "\n", 2)[0]

	fmt.Printf("%v %v %v\n", changeId, c.Hash, short)
}

func main() {
	CheckArgs("<commit>")
	commit := os.Args[1]

	r, err := git.PlainOpen(".")
	CheckIfError(err)

	refHEAD, err := r.Head()
	CheckIfError(err)
	hashHEAD := refHEAD.Hash()

	var refOther *plumbing.Reference
	// trying reference name as is
	refOther, err = r.Reference(plumbing.ReferenceName(commit), true)
	if err != nil {
		// expanding short reference name to full name
		for _, format := range plumbing.RefRevParseRules {
			full := fmt.Sprintf(format, commit)
			refOther, err = r.Reference(plumbing.ReferenceName(full), true)
			if err == nil {
				break
			}
		}
	}
	CheckIfError(err)

	// read HEAD
	mapHEAD, _, err := GerritCommits(r, hashHEAD)
	CheckIfError(err)

	// read other
	hashOther := refOther.Hash()
	mapOther, listOther, err := GerritCommits(r, hashOther)

	// remove commits that already in HEAD
	for k, _ := range mapHEAD {
		delete(mapOther, k)
	}

	// print left commits in other branch
	for e := listOther.Front(); e != nil; e = e.Next() {
		var changeId string = e.Value.(string)
		if c := mapOther[changeId]; c != nil {
			PrintCommit(c, changeId)
		}
		
	}
	
}
