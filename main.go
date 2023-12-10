package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var metadata *bool

func fetch(link string, filename string) error {
	fmt.Println("Fetching ", link)
	client := http.DefaultClient
	client.Timeout = time.Second * 10
	//not using client.Get in order to support custom http headers and methods if needed in the future
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return err
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	took := time.Now().Sub(start)

	//preferred way would have been to use io.Copy but I need to read the body twice to extract metadata
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if *metadata {
		fmt.Println("Calling: ", link)
		fmt.Println("Took: ", took.Milliseconds(), " milliseconds")
		// a more scalable solution would be to use something like
		// golang.org/x/net/html to parse the actual page and retrieve just the tags we're interested in
		linksnum := bytes.Count(body, []byte("href"))
		fmt.Println("Links: ", linksnum)
		if err != nil {
			return err
		}
		imgnum := bytes.Count(body, []byte("<img"))
		fmt.Println("Images: ", imgnum)
		fmt.Println("---------------------------------------")
	}
	if err = os.WriteFile(filename, body, 0644); err != nil {
		return err
	}
	return nil

}

func main() {
	metadata = flag.Bool("metadata", false, "show metadata for calls")
	flag.Parse()
	var links []string
	for i := 0; i < flag.NArg(); i++ {
		links = append(links, flag.Arg(i))
	}
	fmt.Println(links)
	if *metadata {
		fmt.Println("metadata")
	}
	fmt.Println(len(links))
	for _, link := range links {
		//If I wanted to support automatic content-type detection I could
		//either use http.DetectContentType or just read the header, it seemed out of scope
		filename := strings.TrimPrefix(link, "https://")
		filename = strings.TrimPrefix(filename, "http://")
		fmt.Println(filename)
		if err := fetch(link, filename+".html"); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}
}
