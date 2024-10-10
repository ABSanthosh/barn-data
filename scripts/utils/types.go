package utils

type Topic struct {
	Category string `json:"category"`
	Title    string `json:"title"`
	XmlUrl   string `json:"xmlUrl"`
	Desc     string `json:"desc"`
	Type     string `json:"type"`
}

type Recommended map[string][]Topic

type FeedItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Link        string   `json:"link"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Published   string   `json:"published"`
	Image       string   `json:"image"`
	Source      string   `json:"source"`
	Categories  []string `json:"categories"`
}
