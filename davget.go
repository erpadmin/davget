package davget

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
type Multistatus struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []Response `xml:"response"`
}

type Response struct {
	Href     string     `xml:"href"`
	Propstat []Propstat `xml:"propstat"`
}

type Propstat struct {
	Prop   Prop   `xml:"prop"`
	Status string `xml:"status"`
}

type Prop struct {
	DisplayName  string       `xml:"displayname"`
	ResourceType ResourceType `xml:"resourcetype"`
}

// PROPFIND request body for displayname and resourcetype
var propfindBody = `<?xml version="1.0" encoding="utf-8" ?>
<D:propfind xmlns:D="DAV:">
  <D:prop>
    <D:displayname/>
    <D:resourcetype/>
  </D:prop>
</D:propfind>`

type ResourceType struct {
	Collection *struct{} `xml:"collection"`
}

// call PROPFIND against URL and convert to typical HTTP GET response
func List(urlStr string) (string, error) {
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

	var ms Multistatus
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
func GetRecursive(urlStr string) {
	fmt.Println("GetRecursive called with URL:", urlStr)

	req, err := http.NewRequest("PROPFIND", urlStr, bytes.NewBufferString(propfindBody))
	if err != nil {
		fmt.Printf("Error creating PROPFIND request: %v\n", err)
		return
	}
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error performing PROPFIND: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 207 {
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var ms Multistatus
	if err := xml.Unmarshal(data, &ms); err != nil {
		fmt.Printf("Error unmarshalling XML: %v\n", err)
		return
	}

	reqURL, err := url.Parse(urlStr)
	if err != nil {
		fmt.Printf("Error parsing URL: %v\n", err)
		return
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
			GetRecursive(r.Href)
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
}
