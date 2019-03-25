package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/RasmusLindroth/indy/pkg/news"

	//include mysql-driver
	_ "github.com/go-sql-driver/mysql"
)

//Handler interacts with the database
type Handler struct {
	db *sql.DB
}

//New creates a new instance of the Handler struct
func New(user, pass, port string) (*Handler, error) {
	handler := &Handler{}
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

//GetNews returns number of comments for authors during that period with limit
func (handler *Handler) GetNews(lowLimit, highLimit int) ([]*news.Article, error) {

	query := fmt.Sprintf("SELECT `id`, `site`, `title`, `content`, `url`, `matches`, `date` FROM news ORDER BY date DESC LIMIT %d, %d",
		lowLimit, highLimit)

	var articles []*news.Article

	err := handler.makeQuery(query, func(rows *sql.Rows) error {
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
