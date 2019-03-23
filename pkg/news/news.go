package news

import (
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

//Site describes one RSS-feed
type Site struct {
	ID        uint   `yaml:"id"`
	URL       string `yaml:"url"`
	Name      string `yaml:"name"`
	MustMatch bool   `yaml:"strict"`
}

//Article holds an news article
type Article struct {
	ID      uint
	Site    uint
	Title   string
	Content string
	URL     string
	Matches ArticleMatch
	Date    time.Time
}

//NewArticle returns a new article from a feed item
func NewArticle(item *gofeed.Item, site *Site) *Article {
	a := &Article{}
	a.Site = site.ID
	a.Title = item.Title
	a.Content = item.Description
	a.URL = item.Link
	a.Date = *item.PublishedParsed
	a.Matches = a.matchingArticle(site)

	return a
}

//ArticleMatch is used to hold match result
type ArticleMatch uint8

const (
	//MarcusMatch if the article is about Ericsson
	MarcusMatch ArticleMatch = 1 << iota
	//FelixMatch if the article is about Rosenqvist
	FelixMatch
	//IndyMatch if the article is about IndyCar
	IndyMatch
)

//Word holds a matching word
type Word struct {
	Word   string
	Strict bool
}

//MatchWords holds words to match against an article
type MatchWords struct {
	Bit   ArticleMatch
	Words []Word
}

//Match checks if an article contains ArticleMatch
func (article *Article) Match(match ArticleMatch) bool {
	return article.Matches&match != 0
}

func (article *Article) ContainsMarcus() bool {
	return article.Match(MarcusMatch)
}

func (article *Article) ContainsFelix() bool {
	return article.Match(FelixMatch)
}

func (article *Article) matchingArticle(site *Site) ArticleMatch {
	matchWords := []MatchWords{
		MatchWords{
			Bit: MarcusMatch,
			Words: []Word{
				Word{Word: "marcus ericsson", Strict: true},
				Word{Word: "ericsson", Strict: false},
				Word{Word: "marcus", Strict: false},
			},
		},
		MatchWords{
			Bit: FelixMatch,
			Words: []Word{
				Word{Word: "felix rosenqvist", Strict: true},
				Word{Word: "felix", Strict: false},
				Word{Word: "rosenqvist", Strict: false},
			},
		},
		MatchWords{
			Bit: IndyMatch,
			Words: []Word{
				Word{Word: "indycar", Strict: true},
			},
		},
	}

	var match ArticleMatch

	title := strings.ToLower(article.Title)
	text := strings.ToLower(article.Content)

	for _, words := range matchWords {
		for _, w := range words.Words {
			if site.MustMatch == true && w.Strict == false {
				continue
			}
			if strings.Contains(title, w.Word) || strings.Contains(text, w.Word) {
				match |= words.Bit
				break
			}
		}
	}

	return match
}
