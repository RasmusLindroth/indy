package fetcher

import (
	"log"
	"sync"

	"github.com/RasmusLindroth/indy/pkg/database"
	"github.com/RasmusLindroth/indy/pkg/news"
	"github.com/go-sql-driver/mysql"
	"github.com/mmcdole/gofeed"
)

var parser = gofeed.NewParser()
var cache = &cacheHanddler{Mutex: &sync.Mutex{}}

type cacheHanddler struct {
	Mutex *sync.Mutex
	URLs  []string
}

func (c *cacheHanddler) exists(article *news.Article) bool {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	for _, url := range c.URLs {
		if article.URL == url {
			return true
		}
	}

	return false
}

func (c *cacheHanddler) add(article *news.Article) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if len(c.URLs) > 500 {
		c.URLs = append(c.URLs[:0], c.URLs[1:]...)
	}

	c.URLs = append(c.URLs, article.URL)
}

//FetchSites gets all feeds and inserts them to the database
func FetchSites(sites []*news.Site, db *database.Handler) {
	for _, site := range sites {
		go fetchSite(site, db)
	}
}

func fetchSite(site *news.Site, db *database.Handler) {
	feed, err := parser.ParseURL(site.URL)
	if err != nil {
		//Log this
		return
	}

	for _, item := range feed.Items {
		article := news.NewArticle(item, site)

		if article.Matches == 0 && site.MustMatch == true {
			continue
		}

		if cache.exists(article) {
			continue
		}

		err = insertItem(article, site, db)
		if err != nil {
			log.Printf("Insert err: %v\n", err)
		}
	}
}

func insertItem(article *news.Article, site *news.Site, db *database.Handler) error {
	err := db.AddNews(article)

	if err != nil {
		me, ok := err.(*mysql.MySQLError)
		if ok && me.Number == 1062 {
			cache.add(article)
			return nil
		}
	} else {
		cache.add(article)
	}

	return err
}
