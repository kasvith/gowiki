package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Page struct {
	Title string
	Body  []byte
}

type HomePage struct {
	Indexes map[string]string
}

const DATAPATH = "data/"
const TMPLPATH = "tmpl/"

var views = []string{
	"home.html",
	"edit.html",
	"view.html",
	"new.html",
}

func getAbsViewPath() []string {
	for i := range views {
		views[i] = TMPLPATH + views[i]
	}

	return views
}

var wikipages = make(map[string]string, 0)
var templates = template.Must(template.ParseFiles(getAbsViewPath()...))
var validPath = regexp.MustCompile(`^/(edit|save|view)/([a-zA-Z0-9\s]+)$`)

func buildIndex() {
	// build index
	files, err := ioutil.ReadDir(DATAPATH)
	if err != nil {
		log.Println(err)
	}

	for _, f := range files {
		file := strings.Split(f.Name(), ".")
		wikipages[f.Name()] = file[0]
	}

	log.Println("Indexed files")
}

func (p *Page) save() error {
	filename := DATAPATH + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	fmt.Println(title)
	filename := DATAPATH + title + ".txt"
	body, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	return &Page{title, body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/static/404.html", http.StatusNotFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go buildIndex()
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	home := &HomePage{wikipages}
	renderTemplate(w, "home", home)
}

func newWikiHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "new", nil)
}

func makeSaveHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title := r.FormValue("title")
		fmt.Println("title", title)
		if title == "" || len(title) == 0 {
			fmt.Println("redirecting")
			http.Error(w, "Title is required", http.StatusInternalServerError)
			return
		}

		fn(w, r, title)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)

	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	buildIndex()

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/wiki/new", newWikiHandler)
	http.HandleFunc("/wiki/new/save", makeSaveHandler(saveHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
