package util

import (
	"syscall"
)

// Space returns total and free bytes available in a directory, e.g. `/`.
// Think of it as "df" UNIX command.
func SpaceUnix(path string) (total, free int, err error) {
	s := syscall.Statfs_t{}
	err = syscall.Statfs(path, &s)
	if err != nil {
		return
	}
	total = int(s.Bsize) * int(s.Blocks)
	free = int(s.Bsize) * int(s.Bfree)
	return
}