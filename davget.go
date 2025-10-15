package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
)

// WebDAV PROPFIND response structures
type multiStatus struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []response `xml:"response"`
}

type response struct {
	Href     string     `xml:"href"`
	Propstat []propstat `xml:"propstat"`
}

type propstat struct {
	Prop   prop   `xml:"prop"`
	Status string `xml:"status"`
}

type prop struct {
	DisplayName  string       `xml:"displayname"`
	ResourceType resourceType `xml:"resourcetype"`
}

// PROPFIND request body for displayname and resourcetype
var propfindBody = `<?xml version="1.0" encoding="utf-8" ?>
<D:propfind xmlns:D="DAV:">
  <D:prop>
    <D:displayname/>
    <D:resourcetype/>
  </D:prop>
</D:propfind>`

type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

// call PROPFIND against URL and convert to typical HTTP GET response
func davList(urlStr string) (string, error) {
	req, err := http.NewRequest("PROPFIND", urlStr, bytes.NewBufferString(propfindBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 207 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ms multiStatus
	if err := xml.Unmarshal(data, &ms); err != nil {
		return "", err
	}

	// Parse the requested URL to get its path
	reqURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	reqPath := reqURL.Path

	var result string
	for _, r := range ms.Responses {
		// Parse the response href to get its path
		respURL, err := url.Parse(r.Href)
		if err != nil {
			continue // skip invalid hrefs
		}
		// Skip the requested directory itself
		if respURL.Path == reqPath {
			continue
		}

		name := r.Propstat[0].Prop.DisplayName
		if name == "" {
			name = r.Href
		}
		result += fmt.Sprintf("%s\n", name)
	}

	return result, nil
}

// Recursively download files and directories from a WebDAV server
func davGetRecursive(urlStr string) error {
	fmt.Println("GetRecursive called with URL:", urlStr)

	req, err := http.NewRequest("PROPFIND", urlStr, bytes.NewBufferString(propfindBody))
	if err != nil {
		return fmt.Errorf("error creating PROPFIND request: %v", err)
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error performing PROPFIND: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 207 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	var ms multiStatus
	if err := xml.Unmarshal(data, &ms); err != nil {
		return fmt.Errorf("error unmarshalling XML: %v", err)
	}

	reqURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("error parsing URL: %v", err)
	}
	reqPath := reqURL.Path

	for _, r := range ms.Responses {
		respURL, err := url.Parse(r.Href)
		if err != nil {
			continue
		}
		if respURL.Path == reqPath {
			continue
		}

		prop := r.Propstat[0].Prop
		name := prop.DisplayName
		if name == "" {
			name = path.Base(respURL.Path)
		}

		if prop.ResourceType.Collection != nil {
			fmt.Printf("Create and enter directory: %s\n", name)
			os.Mkdir(name, 0755) // create local directory
			os.Chdir(name)       // change into it
			davGetRecursive(r.Href)
			os.Chdir("..") // go back up
		} else {
			fmt.Printf("Downloading file: %s\n", name)
			// r.Href is not the full URL
			fileURL := reqURL.Scheme + "://" + reqURL.Host + r.Href
			fileResp, err := client.Get(fileURL)
			if err != nil {
				fmt.Printf("Error downloading file: %v\n", err)
				continue
			}
			defer fileResp.Body.Close()
			outFile, err := os.Create(name)
			if err != nil {
				fmt.Printf("Error creating file: %v\n", err)
				continue
			}
			io.Copy(outFile, fileResp.Body)
			outFile.Close()
		}
	}
	return nil
}
