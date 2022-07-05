package main

import (
	"os"
	"path/filepath"
)

func WalkMatch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func InitFilesInfoMap(matches []string) map[string]os.FileInfo {
	filesInfo := make(map[string]os.FileInfo)
	for _, path := range matches {
		fileInfo, err := os.Stat(path)
		if err == nil {
			filesInfo[path] = fileInfo
		}
	}

	return filesInfo
}