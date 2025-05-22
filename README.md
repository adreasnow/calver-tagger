# calver-tagger

Iterates through all the tags beginning with `v` in the specified repo and creates a CalVer tag on the same commit.

## Usage

Basic usage, creates new CalVer tags and leaves the old tags:

```
go run . -path ./path/to/repo/to/tag
```

- Append `-delete` to delete the old tags
- Append `-dry-run` to print the plan and exit

All new CalVer tags will have the old tag as the message, along with any message that existed on the source tag.
