package main

import (
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestGetAllTags(t *testing.T) {
	body := `<html>
  <body>
  <a href="#hello">I am a link </a>
  <a href="#hello">I am another  link </a>
  <a href="#hello">I am a third link </a>
  <ul>
  <li>1</li>
  <li><a> I am the last link </a></li>
  </ul>
  </body>
  </html>`
	root, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal("Failed with ", err.Error())
	}
	as := getAllTags(root, "a")
	lis := getAllTags(root, "li")
	t.Log(as, lis)
	if len(as) != 4 {
		t.Fatalf("Expected %d <a> tags, got %d\n", 4, len(as))
	}
	f := as[0]
	t.Log(f.Data, f.Attr)
	if len(lis) != 2 {
		t.Fatalf("Expected %d <a> tags, got %d\n", 2, len(lis))
	}
}
