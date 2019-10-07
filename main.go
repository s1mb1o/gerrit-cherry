package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	//	"time"

	"gopkg.in/src-d/go-git.v4"
	. "gopkg.in/src-d/go-git.v4/_examples"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const kChangeId = "Change-Id: "

func GerritCommits(r *git.Repository, hash plumbing.Hash) (map[string]object.Commit, error) {
	var m map[string]object.Commit = make(map[string]object.Commit)
	var err error

	cIter, err := r.Log(&git.LogOptions{From: hash})
	if err != nil {
		return nil, err
	}

	err = cIter.ForEach(func(c *object.Commit) error {
		scanner := bufio.NewScanner(strings.NewReader(fmt.Sprintf("%v", c)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, kChangeId) {
				changeId := strings.TrimPrefix(line, kChangeId)

				m[changeId] = *c
				break
			}
		}
		return nil
	})

	return m, nil
}

func main() {
	CheckArgs("<commit>")
	commit := os.Args[1]

	r, err := git.PlainOpen(".")
	CheckIfError(err)

	refHEAD, err := r.Head()
	CheckIfError(err)
	hashHEAD := refHEAD.Hash()
	mapHEAD, err := GerritCommits(r, hashHEAD)
	CheckIfError(err)

	var refOther *plumbing.Reference
	// trying reference name as is
	refOther, err = r.Reference(plumbing.ReferenceName(commit), true)
	if err != nil {
		// expanding short reference name to full name
		for _, format := range plumbing.RefRevParseRules {
			full := fmt.Sprintf(format, commit)
			fmt.Println(full)
			refOther, err = r.Reference(plumbing.ReferenceName(full), true)
			if err == nil {
				break
			}
		}
	}
	CheckIfError(err)

	hashOther := refOther.Hash()
	mapOther, err := GerritCommits(r, hashOther)

	// remove commits that already in HEAD
	for k, _ := range mapHEAD {
		delete(mapOther, k)
	}

	fmt.Println(mapOther)
}
