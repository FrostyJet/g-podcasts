package models

type Podcast struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Poster      string   `json:"poster"`
	Tracks      []string `json:"tracks"`
	OriginalUrl string   `json:"original_url"`
}
