package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"slices"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var repo = flag.String("path", "", "The path to the repo to re-tag")
var dryRun = flag.Bool("dry-run", false, "Whether to only display the changes without tagging")
var deleteOld = flag.Bool("delete", false, "Whether to delete the old tags")

type tag struct {
	from    string
	to      string
	tag     *plumbing.Reference
	message string
	commit  *object.Commit
}

func main() {
	flag.Parse()

	if *repo == "" {
		log.Fatal("must specify path to repo with -path")
	}

	path := path.Clean(*repo)
	_, err := os.ReadDir(path)
	if err != nil {
		log.Fatalf("could not read path '%s': %v ", path, err)
	}

	repo, err := git.PlainOpen(path)
	if err != nil {
		log.Fatalf("could not load git repo from path '%s': %v ", path, err)
	}

	tags, err := getTags(repo)
	if err != nil {
		log.Fatalf("could not get tags from repo: %v", err)
	}

	if len(tags) == 0 {
		log.Println("no tags to update")
	}

	fmt.Println("Tags to migrate:")
	for _, tag := range tags {
		fmt.Printf("%s --> %s message: %s\n", tag.from, tag.to, tag.message)
	}

	if *dryRun {
		return
	}

	for _, tag := range tags {
		_, err = repo.CreateTag(tag.to, tag.commit.Hash, &git.CreateTagOptions{
			Message: tag.message,
		})
		if err != nil {
			log.Printf("could not create tag %s on commit %s: %v", tag.to, tag.commit.Hash.String(), err)
		}

		if !*deleteOld {
			continue
		}

		err := repo.DeleteTag(tag.from)
		if err != nil {
			log.Printf("could not get delete tag %s: %v", tag.from, err)
		}
	}
}

func getTags(repo *git.Repository) (out []*tag, err error) {
	tags := []*tag{}

	tagIter, err := repo.Tags()
	if err != nil {
		err = fmt.Errorf("could not read tag iterator from repo: %w ", err)
		return
	}

	err = tagIter.ForEach(func(tagOB *plumbing.Reference) error {
		tags = append(tags, &tag{
			from:    tagOB.Name().Short(),
			tag:     tagOB,
			message: "converted from tag " + tagOB.Name().Short(),
		})
		return nil
	})

	if err != nil {
		err = fmt.Errorf("error iterating over tags: %w ", err)
		return
	}

	for _, tag := range tags {
		tag.commit, err = repo.CommitObject(tag.tag.Hash())
		if err != nil {
			log.Printf("error getting commit for tag %s: %v", tag.from, err)
			continue
		}

		tagObject, err := repo.TagObject(tag.tag.Hash())
		if err != nil && !errors.Is(err, plumbing.ErrObjectNotFound) {
			log.Printf("error getting tag object for tag %s: %v", tag.from, err)
			continue
		}

		if tagObject != nil {
			tag.message = fmt.Sprintf("%s (%s)", tagObject.Message, tag.message)
		}

		commitTime := tag.commit.Committer.When

		year := commitTime.Year()
		month := int(commitTime.Month())
		tag.to = fmt.Sprintf("v%d.%d", year, month)
	}

	tagged := calculateTagRevision(tags)
	out = sortTagsByDate(tagged)

	return
}

func sortTagsByDate(tags []*tag) (out []*tag) {
	slices.SortFunc(tags, func(a *tag, b *tag) int {
		return int(a.commit.Committer.When.Unix() - b.commit.Committer.When.Unix())
	})

	out = tags
	return
}

func calculateTagRevision(tags []*tag) (out []*tag) {
	tagMap := map[string][]*tag{}

	for _, tag := range tags {
		tagMap[tag.to] = append(tagMap[tag.to], tag)
	}

	for _, tagList := range tagMap {
		sortedTags := sortTagsByDate(tagList)

		for n, tag := range sortedTags {
			tag.to += fmt.Sprintf(".%d", n+1)
			out = append(out, tag)
		}
	}

	return
}
