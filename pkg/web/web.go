package web

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
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
	"github.com/tdewolff/minify/v2/js"
)

const articlesPerPage = 30

func minifyCSS(path string) ([]byte, error) {
	var minified []byte

	f, err := os.Open(path)
	if err != nil {
		return minified, err
	}

	unminified, err := ioutil.ReadAll(f)
	if err != nil {
		return minified, err
	}

	m := minify.New()
	m.AddFunc("text/css", css.Minify)

	minified, err = m.Bytes("text/css", unminified)
	if err != nil {
		return minified, err
	}

	return minified, err
}

func minifyJS(path string) ([]byte, error) {
	var minified []byte

	f, err := os.Open(path)
	if err != nil {
		return minified, err
	}

	unminified, err := ioutil.ReadAll(f)
	if err != nil {
		return minified, err
	}

	m := minify.New()
	m.AddFunc("application/javascript", js.Minify)

	minified, err = m.Bytes("application/javascript", unminified)
	if err != nil {
		return minified, err
	}

	return minified, err
}

//New returns a new web handler
func New(db *database.Handler, sites []*news.Site, filesPath string) *Handler {
	smap := make(map[uint]string)

	for _, site := range sites {
		smap[site.ID] = site.Name
	}

	minifiedCSS, err := minifyCSS(filesPath + "/include/style.css")

	if err != nil {
		log.Fatal(err)
	}

	minifiedJS, err := minifyJS(filesPath + "/include/main.js")

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

	templates := make(map[string]*template.Template)

	templates["index"] = template.Must(template.New("").Funcs(funcs).ParseFiles(filesPath+"/templates/base.gohtml", filesPath+"/templates/index.gohtml"))
	templates["error"] = template.Must(template.New("").ParseFiles(filesPath+"/templates/base.gohtml", filesPath+"/templates/error.gohtml"))

	return &Handler{
		DB:        db,
		Templates: templates,
		Sites:     smap,
		filesPath: filesPath,
		css:       template.CSS(string(minifiedCSS)),
		js:        template.JS(string(minifiedJS)),
	}
}

//Handler holds something
type Handler struct {
	DB        *database.Handler
	Templates map[string]*template.Template
	Sites     map[uint]string
	filesPath string
	css       template.CSS
	js        template.JS
}

//StartServer starts the server
func (handler *Handler) StartServer(port string) {
	r := mux.NewRouter()
	r.HandleFunc("/", handler.HomeHandler)
	r.HandleFunc("/news/page/{page:[0-9]+}", handler.HomeHandler)
	r.HandleFunc("/sitemap.xml", handler.SiteMap)
	r.HandleFunc("/manifest.json", handler.Manifest)
	r.HandleFunc("/robots.txt", handler.Robots)
	r.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, handler.filesPath+"/static/sw.js")
	})
	r.HandleFunc("/error/404", func(w http.ResponseWriter, r *http.Request) {
		handler.errorHandler(w, r, http.StatusNotFound)
	})
	r.HandleFunc("/error/offline", func(w http.ResponseWriter, r *http.Request) {
		handler.errorHandler(w, r, http.StatusOK)
	})
	r.PathPrefix("/images/").Handler(http.StripPrefix("/images/", http.FileServer(http.Dir(handler.filesPath+"/static/images"))))

	r.NotFoundHandler = http.HandlerFunc(handler.NotFound)
	gzip := handlers.CompressHandler(r)

	http.ListenAndServe(":"+port, handlers.RecoveryHandler()(gzip))
}

//ErrorData holds data to serve
type ErrorData struct {
	Title      string
	ErrorTitle string
	ErrorMsg   string
	CSS        template.CSS
	JS         template.JS
	Canonical  bool
}

//NotFound runs the errorHandler
func (handler *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	handler.errorHandler(w, r, http.StatusNotFound)
}

func (handler *Handler) errorHandler(w http.ResponseWriter, r *http.Request, statusCode int) {
	/*
		Apparently we need to set the content type.
		https://github.com/gorilla/handlers/issues/83#issuecomment-244800033
	*/
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	var title string
	var msg string

	switch statusCode {
	//Used for offline requests
	case 200:
		title = "Du är offline"
		msg = "Tyvärr har du ingen internetuppkoppling nu. Försök igen senare."
	case 404:
		title = "Sidan finns inte (404)"
		msg = "Tyvärr har du råkat hamna på en sida som inte finns."
	case 500:
		title = "Ett fel i servern har inträffat (500)"
		msg = "Tyvärr har något gått snett i servern. Var vänlig och försök igen."
	case 401:
		title = "Du har inte rätt behörighet (401)"
		msg = "Du måste logga in för att besöka den här sidan."
	}

	data := ErrorData{
		Title:      fmt.Sprintf("IndyCar - %d", statusCode),
		ErrorTitle: title,
		ErrorMsg:   msg,
		CSS:        handler.css,
		JS:         handler.js,
		Canonical:  false,
	}

	handler.Templates["error"].ExecuteTemplate(w, "base", data)
}

//HomeData holds data to serve
type HomeData struct {
	Title     string
	Articles  []*news.Article
	Sites     map[uint]string
	CSS       template.CSS
	JS        template.JS
	PageNow   int
	PageTotal int
	Canonical bool
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

	data := HomeData{Title: "IndyCar - Sverige", Sites: handler.Sites, CSS: handler.css, JS: handler.js}
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
	data.Canonical = r.URL.Path != "/" && page == 1

	handler.Templates["index"].ExecuteTemplate(w, "base", data)
}

//SiteMapURLSet holds an URL set
type SiteMapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []SiteMapURL `xml:"url"`
}

//SiteMapURL holds an URL
type SiteMapURL struct {
	Loc string `xml:"loc"`
}

//SiteMap handles sitemap requests
func (handler *Handler) SiteMap(w http.ResponseWriter, r *http.Request) {
	count, err := handler.DB.NumArticles()

	if err != nil {
		handler.errorHandler(w, r, http.StatusInternalServerError)
		log.Printf("Error getting article count: %v\n", err)
		return
	}

	pages := int(math.Ceil(float64(count) / float64(articlesPerPage)))

	var urls []SiteMapURL
	for i := 1; i <= pages; i++ {
		if i == 1 {
			urls = append(urls, SiteMapURL{Loc: "https://indycar.xyz/"})
		} else {
			urls = append(urls, SiteMapURL{Loc: fmt.Sprintf("https://indycar.xyz/news/page/%d", i)})
		}
	}
	smap := SiteMapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	data, err := xml.MarshalIndent(smap, "", "    ")
	if err != nil {
		handler.errorHandler(w, r, http.StatusInternalServerError)
		log.Printf("Error generating sitemap: %v\n", err)
		return
	}

	fmt.Fprintf(w, "%s%s", xml.Header, data)
}

//Manifest holds the data for manifest.json
type Manifest struct {
	ShortName       string         `json:"short_name"`
	Name            string         `json:"name"`
	Icons           []ManifestIcon `json:"icons"`
	StartURL        string         `json:"start_url"`
	BackgroundColor string         `json:"background_color"`
	Display         string         `json:"display"`
	Scope           string         `json:"scope"`
	ThemeColor      string         `json:"theme_color"`
}

//ManifestIcon holds one icon in manifest
type ManifestIcon struct {
	Src   string `json:"src"`
	Type  string `json:"type"`
	Sizes string `json:"sizes"`
}

//Manifest handles manifest.json requests
func (handler *Handler) Manifest(w http.ResponseWriter, r *http.Request) {
	manifest := Manifest{
		ShortName: "IndySwe",
		Name:      "Indycar Sverige",
		Icons: []ManifestIcon{
			ManifestIcon{
				Src:   "/images/indy192.png",
				Type:  "image/png",
				Sizes: "192x192",
			},
			ManifestIcon{
				Src:   "/images/indy512.png",
				Type:  "image/png",
				Sizes: "512x512",
			},
		},
		StartURL:        "/",
		BackgroundColor: "#006aa7",
		Display:         "standalone",
		Scope:           "/",
		ThemeColor:      "#006aa7",
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		handler.errorHandler(w, r, http.StatusInternalServerError)
		log.Printf("Error generating manifest: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

//Robots handles robots.txt requests
func (handler *Handler) Robots(w http.ResponseWriter, r *http.Request) {
	sitemap := "https://indycar.xyz/sitemap.xml"

	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /error/offline\n\nSitemap: %s", sitemap)
}
