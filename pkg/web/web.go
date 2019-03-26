package web

import (
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/RasmusLindroth/indy/pkg/database"
	"github.com/RasmusLindroth/indy/pkg/news"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
)

const articlesPerPage = 30

//New returns a new web handler
func New(db *database.Handler, sites []*news.Site, filesPath string) *Handler {
	smap := make(map[uint]string)

	for _, site := range sites {
		smap[site.ID] = site.Name
	}

	f, err := os.Open(filesPath + "/static/style.css")
	if err != nil {
		log.Fatal(err)
	}

	unminified, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	m := minify.New()
	m.AddFunc("text/css", css.Minify)

	minified, err := m.Bytes("text/css", unminified)
	if err != nil {
		log.Fatal(err)
	}

	funcs := template.FuncMap{
		"pagelist": func(pages int) []int {
			var res []int
			for i := 1; i <= pages; i++ {
				res = append(res, i)
			}
			return res
		},
	}

	templates := template.Must(template.New("").Funcs(funcs).ParseFiles(filesPath + "/templates/index.gohtml"))

	return &Handler{
		DB:        db,
		Templates: templates,
		Sites:     smap,
		filesPath: filesPath,
		css:       template.CSS(string(minified)),
	}
}

//Handler holds something
type Handler struct {
	DB        *database.Handler
	Templates *template.Template
	Sites     map[uint]string
	filesPath string
	css       template.CSS
}

//StartServer starts the server
func (handler *Handler) StartServer(port string) {
	r := mux.NewRouter()
	r.HandleFunc("/", handler.HomeHandler)
	r.HandleFunc("/news/page/{page:[0-9]+}", handler.HomeHandler)
	gzip := handlers.CompressHandler(r)

	http.ListenAndServe(":"+port, handlers.RecoveryHandler()(gzip))
}

//HomeData holds data to serve
type HomeData struct {
	Title     string
	Articles  []*news.Article
	Sites     map[uint]string
	CSS       template.CSS
	PageNow   int
	PageTotal int
}

func (handler *Handler) errorHandler(w http.ResponseWriter, r *http.Request, statusCode int) {
	http.Error(w, http.StatusText(statusCode), statusCode)
}

//HomeHandler handles home requests
func (handler *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	var page = 1
	vars := mux.Vars(r)

	if val, ok := vars["page"]; ok {
		num, err := strconv.Atoi(val)

		if err != nil {
			handler.errorHandler(w, r, http.StatusNotFound)
			return
		}

		if num < 1 {
			handler.errorHandler(w, r, http.StatusNotFound)
			return
		}

		page = num
	}

	data := HomeData{Title: "IndyCar - Sverige", Sites: handler.Sites, CSS: handler.css}
	articles, err := handler.DB.GetNews(page, articlesPerPage)
	if err != nil {
		handler.errorHandler(w, r, http.StatusInternalServerError)
		log.Printf("Error serving articles: %v\n", err)
		return
	}

	if len(articles) == 0 {
		handler.errorHandler(w, r, http.StatusNotFound)
		return
	}
	data.Articles = articles

	count, err := handler.DB.NumArticles()

	if err != nil {
		handler.errorHandler(w, r, http.StatusInternalServerError)
		log.Printf("Error getting article count: %v\n", err)
		return
	}

	pages := int(math.Ceil(float64(count) / float64(articlesPerPage)))

	data.PageNow = page
	data.PageTotal = pages

	handler.Templates.ExecuteTemplate(w, "index.gohtml", data)
}
