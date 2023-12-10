package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var showMetadata bool

type Metadata struct {
	links  int
	images int
	host   string
	took   time.Duration
}

func fetch(link string, filename string) (*Metadata, error) {
	client := http.DefaultClient
	client.Timeout = time.Second * 10
	//not using client.Get in order to support custom http headers and methods if needed in the future
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	took := time.Now().Sub(start)

	//preferred way would have been to use io.Copy but I need to read the body twice to extract metadata
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var meta *Metadata
	if showMetadata {

		// golang.org/x/net/html to parse the actual page and retrieve just the tags we're interested in
		// a more scalable solution would be to use something like
		linksnum := bytes.Count(body, []byte("href"))
		imgnum := bytes.Count(body, []byte("<img"))
		meta = &Metadata{
			links:  linksnum,
			images: imgnum,
			host:   link,
			took:   took,
		}
	}
	if err = os.WriteFile(filename, body, 0644); err != nil {
		return nil, err
	}
	return meta, nil
}

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

func dispatchFetch(link, filename string, outch chan<- Metadata, wg *sync.WaitGroup) {
	defer wg.Done()
	data, err := fetch(link, filename)
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
	flag.BoolVar(&showMetadata, "metadata", false, metaUsage)
	flag.BoolVar(&showMetadata, "m", false, metaUsage+" (shorthand)")

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
		filename := strings.TrimPrefix(link, "https://")
		filename = strings.TrimPrefix(filename, "http://")
		wg.Add(1)
		go dispatchFetch(link, filename+".html", metachannel, &wg)
	}
	//After all the fetches have been performed I can close the metachannel and avoid deadlocking
	wg.Wait()
	close(metachannel)
	//Waiting for buffered metadata to finish printing
	<-donech
}
