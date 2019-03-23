package web

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/RasmusLindroth/indy/pkg/database"
	"github.com/RasmusLindroth/indy/pkg/news"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
)

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

	templates := make(map[string]*template.Template)
	templates["home"] = template.Must(template.ParseFiles(filesPath + "/templates/index.gohtml"))

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
	Templates map[string]*template.Template
	Sites     map[uint]string
	filesPath string
	css       template.CSS
}

//StartServer starts the server
func (handler *Handler) StartServer(port string) {
	r := mux.NewRouter()
	r.HandleFunc("/", handler.HomeHandler)
	gzip := handlers.CompressHandler(r)

	http.ListenAndServe(":"+port, handlers.RecoveryHandler()(gzip))
}

//HomeData holds data to serve
type HomeData struct {
	Title    string
	Articles []*news.Article
	Sites    map[uint]string
	CSS      template.CSS
}

//HomeHandler handles home requests
func (handler *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	data := HomeData{Title: "IndyCar - Sverige", Sites: handler.Sites, CSS: handler.css}
	articles, err := handler.DB.GetNews(0, 30)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Printf("Error serving articles: %v\n", err)
		return
	}
	data.Articles = articles

	handler.Templates["home"].Execute(w, data)
}
