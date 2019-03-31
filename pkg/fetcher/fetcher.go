package fetcher

import (
	"fmt"
	"log"

	"github.com/RasmusLindroth/indy/pkg/database"
	"github.com/RasmusLindroth/indy/pkg/news"
	"github.com/go-sql-driver/mysql"
	"github.com/mmcdole/gofeed"
)

var parser = gofeed.NewParser()

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

		existingArticle, err := db.GetItem(article)
		if err != nil {
			log.Printf("Couldn't get article: %v\n", err)
			continue
		}

		if existingArticle != nil {
			err = updateItem(existingArticle, article, db)
			if err != nil {
				log.Printf("Update err: %v\n", err)
			}
			continue
		}

		err = insertItem(article, site, db)
		if err != nil {
			log.Printf("Insert err: %v\n", err)
		}
	}
}

func compareArticles(old, new *news.Article) bool {
	return old.Content != new.Content || old.Date != new.Date || old.Matches != new.Matches ||
		old.Title != new.Title
}

func updateItem(old, new *news.Article, db *database.Handler) error {
	diff := compareArticles(old, new)

	if !diff {
		return nil
	}
	fmt.Println(new.Title)

	return db.UpdateNews(new, old.ID)
}

func insertItem(article *news.Article, site *news.Site, db *database.Handler) error {
	err := db.AddNews(article)

	if err != nil {
		me, ok := err.(*mysql.MySQLError)
		if ok && me.Number == 1062 {
			return nil
		}
	}

	return err
}
