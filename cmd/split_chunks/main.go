package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

const CHUNK_SIZE = 1 * 1024 * 1024 // 1mb
const UPLOAD_DIR = "/media/uploads"

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run cmd/split_chunks/main.go <mp4-file-path> <output-folder-name>")
		return
	}

	mp4Path := os.Args[1]
	outputFolderName := os.Args[2]

	err := os.MkdirAll(filepath.Join(UPLOAD_DIR, outputFolderName), os.ModePerm)
	if err != nil {
		fmt.Printf("error creating upload directory: %v\n", err)
		return
	}

	file, err := os.Open(mp4Path)
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		return
	}
	defer file.Close()

	chunkCount := 0

	for {
		chunkFileName := filepath.Join(UPLOAD_DIR, outputFolderName, strconv.Itoa(chunkCount)+".chunk")

		chunkFile, err := os.Create(chunkFileName)
		if err != nil {
			fmt.Printf("error creating chunk file: %v\n", err)
			return
		}

		_, err = io.CopyN(chunkFile, file, CHUNK_SIZE)
		if err != nil {
			if err == io.EOF {
				chunkFile.Close()
				fmt.Println("finished splitting the file into chunks.")
				break
			} else {
				fmt.Printf("error copying chunk: %v\n", err)
				chunkFile.Close()
				return
			}
		}

		chunkFile.Close()
		fmt.Printf("created chunk: %s\n", chunkFileName)

		chunkCount++
	}
}
