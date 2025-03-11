package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
)

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GuiId       string `xml:"guid"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Language    string `xml:"language"`
	Items       []Item `xml:"item"`
}

type RSS struct {
	Channel Channel `xml:"channel"`
	Etag    string
}

func (rss *RSS) ParseURL(ctx context.Context, u string) error {
	r, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		log.Fatal(err)
	}

	if rss.Etag != "" {
		r.Header.Set("Etag", rss.Etag)
	}

	res, err := http.DefaultClient.Do(r)
	defer func(r io.ReadCloser) {
		_ = r.Close()
	}(res.Body)

	if err != nil {
		log.Fatal(err)
	}

	switch res.StatusCode {
	case http.StatusNotModified:
	case http.StatusOK:
		b, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		if err = xml.Unmarshal(b, rss); err != nil {
			log.Fatal(err)
		}
		rss.Etag = res.Header.Get("Etag")
	default:
		return fmt.Errorf("Unexpected http status code %v", res.StatusCode)
	}
	return nil
}

func (r RSS) Items() []Item {
	items := make([]Item, len(r.Channel.Items))
	copy(items, r.Channel.Items)
	return items
}

type EventRequest struct {
	Previous bool `json:"previous"`
}

type EventResponse struct {
	Title     string `json:"title"`
	Link      string `json:"link"`
	Published string `json:"published"`
}

func main() {
	// fmt.Println(rss.Enties())
	lambda.Start(LatestXKCD)
}

func LatestXKCD(ctx context.Context, req EventRequest) (EventResponse, error) {
	resp := EventResponse{Title: "XKCD", Link: "https://xkcd.com"}
	rss := &RSS{}
	if err := rss.ParseURL(context.Background(), "https://xkcd.com/rss.xml"); err != nil {
		return resp, err
	}
	switch items := rss.Items(); {
	case req.Previous && len(items) > 0:
		resp.Link = items[1].Link
		resp.Title = items[1].Title
		resp.Published = items[1].PubDate
	case len(items) > 0:
		resp.Link = items[0].Link
		resp.Title = items[0].Title
		resp.Published = items[0].PubDate
	}
	return resp, nil
}
