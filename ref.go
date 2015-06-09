package main

import (
	"flag"
	"log"
	"time"

	"github.com/libgit2/git2go"
)

var (
	contents = []byte(time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
	fileName = "some_other_file"
	author   = &git.Signature{
		Name:  "Milton",
		Email: "woof@sourcegraph.com",
		When:  time.Now(),
	}
	refName       = "refs/src/dummy"
	commitMessage = "Initial commit"
)

func main() {
	flag.Parse()
	if args := flag.Args(); len(args) >= 2 {
		fileName = args[0]
		contents = []byte(args[1])
		if len(args) == 3 {
			commitMessage = args[2]
		}
	}

	repo, err := git.OpenRepository(".")
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Free()
	idx, err := git.NewIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer idx.Free()

	var parents []*git.Commit
	ref, err := repo.LookupReference(refName)
	if err == nil {
		// If the ref exists, take note of the head commit and update the index
		// to match the ref's tree.
		commit, err := repo.LookupCommit(ref.Target())
		if err != nil {
			log.Fatal(err)
		}
		tree, err := commit.Tree()
		if err != nil {
			log.Fatal(err)
		}
		if err := idx.ReadTree(tree); err != nil {
			log.Fatal(err)
		}
		parents = []*git.Commit{commit}
	} else {
		if e, ok := err.(*git.GitError); !ok || (ok && e.Code != git.ErrNotFound) {
			log.Fatal(err)
		}
	}

	// Write the data to the repo's ODB and obtain its ID for it.
	odb, err := repo.Odb()
	if err != nil {
		log.Fatal(err)
	}
	defer odb.Free()
	ws, err := odb.NewWriteStream(int64(len(contents)), git.ObjectBlob)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Free()
	n, err := ws.Write(contents)
	if err != nil {
		log.Fatal(err)
	}
	if n != len(contents) {
		log.Fatalf("failed to write full contents to tree (wrote %db)", n)
	}
	if err := ws.Close(); err != nil {
		log.Fatal(err)
	}

	// Create a new index entry linking the new object's to its path.
	entry := git.IndexEntry{
		Mode: git.FilemodeBlob,
		Uid:  0700,
		Gid:  0700,
		Size: uint32(len(contents)),
		Id:   &ws.Id,
		Path: fileName,
	}
	if err := idx.Add(&entry); err != nil {
		log.Fatal(err)
	}
	// Sync the contents of the index with the repo's ODB.
	oid, err := idx.WriteTreeTo(repo)
	if err != nil {
		log.Fatal(err)
	}

	// Commit the new tree.
	tree, err := repo.LookupTree(oid)
	if err != nil {
		log.Fatal(err)
	}
	_, err = repo.CreateCommit(refName, author, author, commitMessage, tree, parents...)
	if err != nil {
		log.Fatal(err)
	}
}
