package org

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// What Regate (regate.go) checks on disk before D3's merge (regateMerged, in
// regate_merge.go): that nothing git does not see is where the merge writes
// (regateCheckInTheWay). It reads the paths the merge writes that HEAD does
// not track (regateMergeWrites), what is on disk in the way of each
// (regatePathInTheWay), and how what lstat finds is spelled on disk
// (regateSpellingOnDisk).

// regateInTheWayListed is how many of the paths in the way of the merge
// regateCheckInTheWay's refusal names before it says how many more there are.
const regateInTheWayListed = 10

// regateCheckInTheWay refuses t when D3's merge of t's base tip would write
// over something on disk in t.root that git does not see: without a word,
// git merge overwrites an ignored file at a path it writes and removes one in
// a directory it replaces with a file (--no-overwrite-ignore guards a
// fast-forward only), and the clean-tree check (gateDirtyLines) lists no
// ignored file. The paths the merge writes that HEAD does not track are
// regateMergeWrites's; one of them is in the way when regatePathInTheWay
// finds something untracked there. The refusal names what is on disk, as it
// is spelled there, the first regateInTheWayListed of them. With no merge base
// it refuses nothing: git merge refuses unrelated histories itself
// (regateMergeTip). The error is for a git that fails, runs past
// regateGitTimeout, or is ended by a signal, ctx done, a path lstat cannot
// read, and a directory that cannot be listed; nothing is merged or recorded
// then either.
func regateCheckInTheWay(ctx context.Context, t gateTarget, env []string, w gateWording) (string, error) {
	r, err := gateGit(ctx, regatePrefix, t.root, "merge-base", t.head, t.base.sha)
	switch {
	case err != nil:
		return "", err
	case r.code == 1 && strings.TrimSpace(r.stdout) == "":
		return "", nil
	case r.code != 0:
		return "", fmt.Errorf("%sgit merge-base %s %s in %s: exit %d: %s", regatePrefix, t.head, t.base.sha, t.root, r.code, r.stderr)
	}
	writes, err := regateMergeWrites(ctx, t, env, strings.TrimSpace(r.stdout))
	if err != nil {
		return "", err
	}
	var inTheWay []string
	seen := make(map[string]bool)
	for _, p := range writes {
		found, err := regatePathInTheWay(ctx, t.root, p)
		if err != nil {
			return "", err
		}
		if found != "" && !seen[found] {
			seen[found] = true
			inTheWay = append(inTheWay, found)
		}
	}
	if len(inTheWay) == 0 {
		return "", nil
	}
	listed := strings.Join(inTheWay[:min(len(inTheWay), regateInTheWayListed)], ", ")
	if more := len(inTheWay) - regateInTheWayListed; more > 0 {
		listed += fmt.Sprintf(", and %d more", more)
	}
	return w.prefix + fmt.Sprintf("merging origin's tip %s of %s into %s in %s would write over what is on disk at %s, which "+
		"HEAD does not track: untracked files, or directories with untracked files in them, which the clean-tree check passes "+
		"over as ignored and git merge overwrites without a word (a file where the merge needs a directory too), so regate "+
		"merged nothing and recorded nothing: move them out of the worktree and %s",
		t.base.sha, t.base.branch, t.branch, t.root, listed, w.again), nil
}

// regateMergeWrites returns the paths D3's merge of t's base tip into HEAD
// writes in t.root that HEAD does not have, mergeBase being HEAD's and the
// tip's merge base. They are what the tree of `git merge-tree --write-tree
// --no-messages HEAD <tip>` adds to HEAD, `git diff --name-only -z
// --no-renames --diff-filter=A HEAD <tree>`; merge-tree prints the tree on
// its first line both for a clean merge (exit 0) and for one that conflicts
// (exit 1). merge-tree runs the merge git merge runs: with env as the merge
// gets it (regateMergeEnv) and no setting changed, so it reads the merge's
// config and attributes, and with HEAD and the tip named as git merge names
// them. Its tree then has what a directory rename moves into the renamed
// directory, and the file of a file/directory conflict under the name git
// merge gives it (<path>~HEAD, <path>~<tip>). It runs in a process group of
// its own, as git var does (regateCheckIdentity). A git whose merge-tree has
// no --write-tree (before git 2.38: `git merge-tree -h` does not name it, the
// way scripts/secret-scan-branch.sh tells) falls back to the paths the tip's
// side changed since mergeBase, `git diff --name-only -z --no-renames
// <mergeBase> <tip>`, that HEAD does not have, `git diff --name-only -z
// --no-renames --diff-filter=A HEAD <tip>`, which misses a directory rename's
// moves. Either way, a path HEAD deleted that the tip did not touch stays
// deleted and is not among them. Any other exit of merge-tree is an error.
func regateMergeWrites(ctx context.Context, t gateTarget, env []string, mergeBase string) ([]string, error) {
	nul := func(c rune) bool { return c == 0 }
	args := []string{"merge-tree", "--write-tree", "--no-messages", "HEAD", t.base.sha}
	r, timedOut, err := regateBoundedGit(ctx, t.root, regateMergeEnv(env), true, args...)
	switch {
	case err != nil:
		return nil, err
	case timedOut:
		return nil, fmt.Errorf("%sgit %s in %s ran past %s and was stopped, so nothing was merged or recorded", regatePrefix,
			strings.Join(args, " "), t.root, regateGitTimeout)
	case r.code == 0, r.code == 1:
		tree, _, _ := strings.Cut(r.stdout, "\n")
		if tree == "" {
			return nil, fmt.Errorf("%sgit %s in %s: exit %d and no tree on its first line", regatePrefix, strings.Join(args, " "),
				t.root, r.code)
		}
		added, err := gateGitOutput(ctx, regatePrefix, t.root, "diff", "--name-only", "-z", "--no-renames", "--diff-filter=A", "HEAD",
			tree)
		return strings.FieldsFunc(added, nul), err
	}
	help, err := gateGit(ctx, regatePrefix, t.root, "merge-tree", "-h")
	if err != nil {
		return nil, err
	}
	if strings.Contains(help.stdout+help.stderr, "--write-tree") {
		return nil, fmt.Errorf("%sgit %s in %s: exit %d: %s", regatePrefix, strings.Join(args, " "), t.root, r.code, r.stderr)
	}
	changed, err := gateGitOutput(ctx, regatePrefix, t.root, "diff", "--name-only", "-z", "--no-renames", mergeBase, t.base.sha)
	if err != nil {
		return nil, err
	}
	added, err := gateGitOutput(ctx, regatePrefix, t.root, "diff", "--name-only", "-z", "--no-renames", "--diff-filter=A", t.head,
		t.base.sha)
	if err != nil {
		return nil, err
	}
	onTipSide := make(map[string]bool)
	for _, p := range strings.FieldsFunc(changed, nul) {
		onTipSide[p] = true
	}
	var writes []string
	for _, p := range strings.FieldsFunc(added, nul) {
		if onTipSide[p] {
			writes = append(writes, p)
		}
	}
	return writes, nil
}

// regatePathInTheWay is what is on disk in root in the way of the merge
// writing p, a path HEAD does not track, relative to root as git names it,
// with slashes; "" for nothing. On a file system that does not tell case
// apart (macOS's default), lstat finds at p, or at a directory above it, what
// is spelled otherwise on disk: lstat of README.md finds readme.md. So what
// lstat finds is judged under its spelling on disk (regateSpellingOnDisk), and
// that is the path returned. It is a non-directory at p (a file, a symlink),
// unless it is spelled otherwise and HEAD tracks it under that spelling (it is
// in the index, which is HEAD's in the clean tree Regate checked): its content
// is then in git and the merge replaces it, as when origin's default branch
// renames readme.md to README.md. It is a directory at p with something
// untracked in it (`git ls-files --others --directory --no-empty-directory`),
// whose files git merge removes when it writes a file there; an empty
// directory, or one whose files HEAD tracks and the merge deletes, is not in
// the way. It is the first path above p that lstat finds as a non-directory
// (a file or a symlink where the merge needs a directory), unless HEAD tracks
// it: the merge then replaces it with the directory. In that clean tree,
// whatever untracked it finds is ignored. Git runs only for a directory at p,
// a non-directory above it, and a non-directory at p spelled otherwise, with
// --literal-pathspecs. The error is for a path lstat cannot read, a directory
// that cannot be listed, a git that fails, and ctx done.
func regatePathInTheWay(ctx context.Context, root, p string) (string, error) {
	parts := strings.Split(p, "/")
	for i := range len(parts) {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(strings.Join(parts[:i+1], "/"))))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return "", nil
		case err != nil:
			return "", fmt.Errorf("%scheck what is on disk in %s where the merge writes %s: %w", regatePrefix, root, p, err)
		case info.IsDir() && i < len(parts)-1:
			continue
		}
		disk, err := regateSpellingOnDisk(root, parts[:i+1])
		if err != nil {
			return "", fmt.Errorf("%scheck what is on disk in %s where the merge writes %s: %w", regatePrefix, root, p, err)
		}
		switch {
		case info.IsDir():
			untracked, err := gateGitOutput(ctx, regatePrefix, root, "--literal-pathspecs", "ls-files", "-z", "--others",
				"--directory", "--no-empty-directory", "--", disk)
			if err != nil || untracked == "" {
				return "", err
			}
		case i < len(parts)-1 || disk != p:
			tracked, err := gateGitOutput(ctx, regatePrefix, root, "--literal-pathspecs", "ls-files", "-z", "--", disk)
			if err != nil || slices.Contains(strings.Split(tracked, "\x00"), disk) {
				return "", err
			}
		}
		return disk, nil
	}
	return "", nil
}

// regateSpellingOnDisk is how the path of parts (its names from root down),
// which lstat finds in root, is spelled on disk, relative to root with
// slashes: each name is its directory's entry of that name, else the first
// entry (in name order) whose name is the same when case is not told apart
// (strings.EqualFold), so on a file system that tells case apart it is the
// path parts spell. A name with no such entry (one the file system finds
// under another Unicode normalization form, say) stays as it is. The error is
// for a directory that cannot be listed.
func regateSpellingOnDisk(root string, parts []string) (string, error) {
	disk := make([]string, 0, len(parts))
	for _, part := range parts {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(strings.Join(disk, "/"))))
		if err != nil {
			return "", err
		}
		i := slices.IndexFunc(entries, func(e fs.DirEntry) bool { return e.Name() == part })
		if i < 0 {
			i = slices.IndexFunc(entries, func(e fs.DirEntry) bool { return strings.EqualFold(e.Name(), part) })
		}
		if i >= 0 {
			part = entries[i].Name()
		}
		disk = append(disk, part)
	}
	return strings.Join(disk, "/"), nil
}
