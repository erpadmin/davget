package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
)

func main() {
	listFlag := flag.Bool("l", false, "List contents of the WebDAV URL")
	recursiveFlag := flag.Bool("r", false, "Recursively download file(s) from the WebDAV URL")
	flag.Parse()

	// Get positional argument (URL)
	if flag.NArg() < 1 {
		fmt.Println("Error: WebDAV URL required as positional argument.")
		return
	}
	urlArg := flag.Arg(0)

	if *listFlag && *recursiveFlag {
		fmt.Println("Error: -l and -r flags are mutually exclusive.")
		return
	}

	if *listFlag {
		output, errL := davList(urlArg)
		if errL != nil {
			fmt.Println("Error listing URL:", errL)
			return
		}
		fmt.Println(output)
	} else if *recursiveFlag {
		err := davGetRecursive(urlArg)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		// Default: download single file
		err := davGetFile(urlArg)
		if err != nil {
			fmt.Println("Error downloading file:", err)
		}
	}
}

// Download a single file from WebDAV
func davGetFile(urlStr string) error {
	resp, err := http.Get(urlStr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	// Use last part of URL as filename
	_, file := path.Split(urlStr)
	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
