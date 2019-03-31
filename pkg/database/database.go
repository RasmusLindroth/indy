package database

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/RasmusLindroth/indy/pkg/news"

	//include mysql-driver
	_ "github.com/go-sql-driver/mysql"
)

//Handler interacts with the database
type Handler struct {
	db    *sql.DB
	count *ArticleCount
}

//New creates a new instance of the Handler struct
func New(user, pass, port string) (*Handler, error) {
	handler := &Handler{count: NewArticleCount()}
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(127.0.0.1:%s)/indy?parseTime=true", user, pass, port))
	if err != nil {
		return handler, err
	}

	err = db.Ping()
	if err != nil {
		return handler, err
	}

	db.SetConnMaxLifetime(10 * time.Second)

	handler.db = db
	return handler, nil
}

//ArticleCount is a cache for the number of articles
type ArticleCount struct {
	mutex   *sync.Mutex
	expires time.Time
	count   int
}

//NewArticleCount returns a populated ArticleCount
func NewArticleCount() *ArticleCount {
	ac := &ArticleCount{
		mutex:   &sync.Mutex{},
		expires: time.Now().Add(time.Duration(-1) * time.Second),
		count:   0,
	}

	return ac
}

//Close closes the database connection
func (handler *Handler) Close() {
	handler.db.Close()
}

//AddNews adds one news article to db
func (handler *Handler) AddNews(article *news.Article) error {
	date := article.Date.UTC().Format("2006-01-02 15:04:05")

	_, err := handler.db.Exec("INSERT INTO news (`site`, `title`, `content`, `url`, `matches`, `date`) VALUES(?, ?, ?, ?, ?, ?)", article.Site, article.Title, article.Content, article.URL, article.Matches, date)

	return err
}

//UpdateNews updates one news article
func (handler *Handler) UpdateNews(article *news.Article, id uint) error {
	date := article.Date.UTC().Format("2006-01-02 15:04:05")

	_, err := handler.db.Exec("UPDATE news SET `title` = ?, `content` = ?, `matches` = ?, `date` = ? WHERE id=?", article.Title, article.Content, article.Matches, date, id)

	return err
}

func (handler *Handler) GetItem(article *news.Article) (*news.Article, error) {

	query := fmt.Sprintf("SELECT `id`, `site`, `title`, `content`, `url`, `matches`, `date` FROM news WHERE `url` = '%s'",
		article.URL)

	var dbArticle *news.Article
	err := handler.makeQuery(query, func(rows *sql.Rows) error {
		tmp := &news.Article{}
		err := rows.Scan(&tmp.ID, &tmp.Site, &tmp.Title, &tmp.Content, &tmp.URL, &tmp.Matches, &tmp.Date)
		if err != nil {

			return err
		}

		dbArticle = tmp
		return nil
	})

	return dbArticle, err
}

type getRow func(row *sql.Rows) error

func (handler *Handler) makeQuery(query string, rowfunc getRow) error {
	rows, err := handler.db.Query(query)

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rowfunc(rows)
		if err != nil {
			return err
		}
	}
	err = rows.Err()
	return err
}

func (handler *Handler) pagination(page int, items int) (int, int, error) {
	if page < 1 {
		return 0, 0, errors.New("Page must be atleast 1")
	}

	start := (page - 1) * items
	stop := items

	return start, stop, nil
}

//GetNews returns number of comments for authors during that period with limit
func (handler *Handler) GetNews(page int, items int) ([]*news.Article, error) {
	var articles []*news.Article

	start, stop, err := handler.pagination(page, items)

	if err != nil {
		return articles, errors.New("Page must be atleast 1")
	}

	query := fmt.Sprintf("SELECT `id`, `site`, `title`, `content`, `url`, `matches`, `date` FROM news ORDER BY date DESC LIMIT %d, %d",
		start, stop)

	err = handler.makeQuery(query, func(rows *sql.Rows) error {
		tmp := &news.Article{}
		err := rows.Scan(&tmp.ID, &tmp.Site, &tmp.Title, &tmp.Content, &tmp.URL, &tmp.Matches, &tmp.Date)
		if err != nil {
			return err
		}

		articles = append(articles, tmp)
		return nil
	})

	return articles, err
}

//NumArticles returns the number of articles
func (handler *Handler) NumArticles() (int, error) {
	handler.count.mutex.Lock()
	defer handler.count.mutex.Unlock()

	now := time.Now()

	if handler.count.expires.After(now) {
		return handler.count.count, nil
	}

	query := "SELECT COUNT(1) as `articles` FROM `news`"

	count := 0
	err := handler.makeQuery(query, func(rows *sql.Rows) error {

		err := rows.Scan(&count)
		if err != nil {
			return err
		}

		return nil
	})

	if err == nil {
		handler.count.count = count
		handler.count.expires = now.Add(5 * time.Minute)
	}

	return count, err
}
