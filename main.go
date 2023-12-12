package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var showMetadata bool
var retrieveAsset bool

type Metadata struct {
	links  int
	images int
	host   string
	took   time.Duration
}

func fetchToFile(uri *url.URL, filename string) ([]byte, error) {
	client := http.DefaultClient
	client.Timeout = time.Second * 10
	//not using client.Get in order to support custom http headers and methods if needed in the future
	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err = os.WriteFile(filename, body, 0644); err != nil {
		return nil, err
	}
	return body, nil

}
func fetch(uri *url.URL, filename string) (*Metadata, error) {
	start := time.Now()
	body, err := fetchToFile(uri, filename)
	if err != nil {
		return nil, err
	}
	took := time.Now().Sub(start)

	//preferred way would have been to use io.Copy but I need to read the body twice to extract metadata

	tree, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var meta *Metadata
	images := getAllTags(tree, "img")
	if retrieveAsset {
		for _, img := range images {
			for _, attr := range img.Attr {
				fmt.Println(attr)
				if attr.Key == "src" {
					//If the href is not a valid url then it means it's a relative path
					nurl, err := url.Parse(attr.Val)
					if err != nil {
						*nurl = *uri
						nurl.Path = attr.Val
					}
					fetchToFile(nurl, attr.Val)
				}
			}
		}

	}

	if showMetadata {
		// a more scalable solution would be to use something like
		// golang.org/x/net/html to parse the actual page and retrieve just the tags we're interested in
		//linksnum := bytes.Count(body, []byte("</a>"))
		linksnum := len(getAllTags(tree, "a"))
		//imgnum := bytes.Count(body, []byte("<img"))
		meta = &Metadata{
			links:  linksnum,
			images: len(images),
			host:   uri.Hostname(),
			took:   took,
		}
	}
	return meta, nil
}

// Walk the html tree to find all occurrences of the specified tag
func getAllTags(node *html.Node, tag string) []*html.Node {
	var res []*html.Node
	nodestack := []*html.Node{node}
	for len(nodestack) > 0 {
		node = nodestack[0]
		nodestack = nodestack[1:]
		if node.FirstChild != nil {
			nodestack = append(nodestack, node.FirstChild)
		}
		if node.NextSibling != nil {
			nodestack = append(nodestack, node.NextSibling)
		}
		if node.Data == tag && node.Type == html.ElementNode {
			res = append(res, node)
		}
	}
	return res
}

// Worker that synchronously prints metadata
func printMetadata(inch <-chan Metadata, donech chan<- bool) {
	for meta := range inch {
		fmt.Println("-------------------------")
		fmt.Println("Calling:", meta.host)
		fmt.Println("Links:", meta.links)
		fmt.Println("Images:", meta.images)
		fmt.Println("Took:", meta.took)
	}
	donech <- true
}

func dispatchFetch(murl *url.URL, filename string, outch chan<- Metadata, wg *sync.WaitGroup) {
	defer wg.Done()
	link := murl.String()
	data, err := fetch(murl, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while fetching %s: %s\n", link, err.Error())
		return
	}
	if data != nil {
		outch <- *data
	}
}
func init() {
	metaUsage := "Show metadata of calls"
	assetUsage := "Retrieve assets together with webpage"
	flag.BoolVar(&showMetadata, "metadata", false, metaUsage)
	flag.BoolVar(&showMetadata, "m", false, metaUsage+" (shorthand)")
	flag.BoolVar(&retrieveAsset, "assets", false, assetUsage)
	flag.BoolVar(&retrieveAsset, "a", false, assetUsage+" (shorthand)")
}

func main() {
	flag.Parse()
	var links []string
	for i := 0; i < flag.NArg(); i++ {
		links = append(links, flag.Arg(i))
	}
	metachannel := make(chan Metadata, 16)
	var wg sync.WaitGroup
	//Another waitgroup would have been just as effective
	donech := make(chan bool)
	go printMetadata(metachannel, donech)
	for _, link := range links {
		//If I wanted to support automatic content-type detection I could
		//either use http.DetectContentType or just read the header, it seemed out of scope
		url, err := url.Parse(link)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid url:", err.Error())
			//Looks more go-ish
			continue
		}
		wg.Add(1)
		filename := url.Host + url.EscapedPath()
		go dispatchFetch(url, filename+".html", metachannel, &wg)
	}
	//After all the fetches have been performed I can close the metachannel and avoid deadlocking
	wg.Wait()
	close(metachannel)
	//Waiting for buffered metadata to finish printing
	<-donech
}
