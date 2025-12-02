package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	tmpl       *template.Template
	artistTmpl *template.Template
	errorTmpl  *template.Template
)

type PageData struct {
	Artists   []Artist
	Locations map[string][]string
	Dates     map[string][]string
	Relation  map[string][]string
	Query     string 
	MembersFilter string

}

type ArtistPageData struct {
	Artist    Artist
	Locations []string
	Dates     []string
	Relation  []string
}

type Artist struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Image      string   `json:"image"`
	FirstAlbum string   `json:"firstAlbum"`
	Members    []string `json:"members"`
}

type LocationsAPI struct {
	Index []struct {
		ID        int      `json:"id"`
		Locations []string `json:"locations"`
	} `json:"index"`
}

type DatesAPI struct {
	Index []struct {
		ID    int      `json:"id"`
		Dates []string `json:"dates"`
	} `json:"index"`
}

type RelationAPI struct {
	Index []struct {
		ID             int               `json:"id"`
		DatesLocations map[string]string `json:"datesLocations"`
	} `json:"index"`
}

const (
	apiArtists   = "https://groupietrackers.herokuapp.com/api/artists"
	apiLocations = "https://groupietrackers.herokuapp.com/api/locations"
	apiDates     = "https://groupietrackers.herokuapp.com/api/dates"
	apiRelation  = "https://groupietrackers.herokuapp.com/api/relation"
)

func main() {

	var err error

	// load index template
	tmpl, err = template.New("index.html").
		Funcs(template.FuncMap{"join": strings.Join}).
		ParseFiles(filepath.Join("templates", "index.html"))
	if err != nil {
		log.Fatalf("Error loading index.html: %v", err)
	}

	// load artist template
	artistTmpl, err = template.New("artist.html").
		Funcs(template.FuncMap{"join": strings.Join}).
		ParseFiles(filepath.Join("templates", "artist.html"))
	if err != nil {
		log.Fatalf("Error loading artist.html: %v", err)
	}

	// load error template
	errorTmpl, err = template.ParseFiles(filepath.Join("templates", "error.html"))
	if err != nil {
		log.Fatalf("Error loading error.html: %v", err)
	}

	// routes
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/artist", handleArtist)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Server running on http://localhost:8080")
	log.Println("Press Ctrl+C to stop the server")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		renderError(w, http.StatusNotFound, "Page Not Found")
		return
	}

	if r.Method != http.MethodGet {
		renderError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	artists, err := fetchArtists()
	if err != nil {
		renderError(w, http.StatusInternalServerError, "Failed to fetch artists")
		return
	}

	query := strings.ToLower(r.URL.Query().Get("q"))
	var filtered []Artist
	if len(query) >= 30 {
		renderError(w, http.StatusBadRequest, "Limit reached") 
	}
	if query != "" {
		for _, a := range artists {
			if strings.Contains(strings.ToLower(a.Name), query) {
				filtered = append(filtered, a)
			}
		}
	} else {
		filtered = artists
	}

	membersFilter := r.URL.Query().Get("members")

	if membersFilter != "" {
		var temp []Artist

		for _, a := range filtered {
			count := len(a.Members)

			switch membersFilter {
			case "1":
				if count == 1 {
					temp = append(temp, a)
				}
			case "2":
				if count == 2 {
					temp = append(temp, a)
				}
			case "3":
				if count == 3 {
					temp = append(temp, a)
				}
			case "4":
				if count == 4 {
					temp = append(temp, a)
				}
			case "5":
				if count >= 5 {
					temp = append(temp, a)
				}
			}
		}

		filtered = temp
	}

	locations, _ := fetchLocations()
	dates, _ := fetchDates()
	relation, _ := fetchRelation()

	pageData := PageData{
		Artists:       filtered,
		Locations:     locations,
		Dates:         dates,
		Relation:      relation,
		Query:         query,
		MembersFilter: membersFilter,
	}

	if err := tmpl.Execute(w, pageData); err != nil {
		renderError(w, http.StatusInternalServerError, "Failed to render template")
	}
}


func handleArtist(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/artist" {
		renderError(w, http.StatusNotFound, "Page Not Found")
		return
	}

	if r.Method != http.MethodGet {
		renderError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		renderError(w, http.StatusBadRequest, "Missing artist id")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		renderError(w, http.StatusBadRequest, "Invalid artist id")
		return
	}

	artists, err := fetchArtists()
	if err != nil {
		renderError(w, http.StatusInternalServerError, "Failed to fetch artists")
		return
	}

	var artist Artist
	found := false
	for _, a := range artists {
		if a.ID == id {
			artist = a
			found = true
			break
		}
	}

	if !found {
		renderError(w, http.StatusNotFound, "Artist not found")
		return
	}

	locationsMap, _ := fetchLocations()
	datesMap, _ := fetchDates()
	relationMap, _ := fetchRelation()

	key := fmt.Sprintf("%d", id)

	data := ArtistPageData{
		Artist:    artist,
		Locations: locationsMap[key],
		Dates:     datesMap[key],
		Relation:  relationMap[key],
	}

	if err := artistTmpl.Execute(w, data); err != nil {
		renderError(w, http.StatusInternalServerError, "Failed to render artist page")
	}
}

func fetchArtists() ([]Artist, error) {
	resp, err := http.Get(apiArtists)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var artists []Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, err
	}
	return artists, nil
}

func fetchLocations() (map[string][]string, error) {
	resp, err := http.Get(apiLocations)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data LocationsAPI
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, entry := range data.Index {
		result[fmt.Sprintf("%d", entry.ID)] = entry.Locations
	}
	return result, nil
}

func fetchDates() (map[string][]string, error) {
	resp, err := http.Get(apiDates)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data DatesAPI
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, entry := range data.Index {
		result[fmt.Sprintf("%d", entry.ID)] = entry.Dates
	}
	return result, nil
}

func fetchRelation() (map[string][]string, error) {
	resp, err := http.Get(apiRelation)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data RelationAPI
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, entry := range data.Index {
		arr := []string{}
		for date, location := range entry.DatesLocations {
			arr = append(arr, fmt.Sprintf("%s → %s", date, location))
		}
		result[fmt.Sprintf("%d", entry.ID)] = arr
	}
	return result, nil
}


type ErrorData struct {
	Code    int
	Title   string
	Message string
}

func renderError(w http.ResponseWriter, code int, msg string) {

	w.WriteHeader(code)

	data := ErrorData{
		Code:    code,
		Message: msg,
	}

	switch code {
	case http.StatusBadRequest:
		data.Title = "400 — Bad Request"
	case http.StatusNotFound:
		data.Title = "404 — Not Found"
	case http.StatusInternalServerError:
		data.Title = "500 — Internal Server Error"
	default:
		data.Title = fmt.Sprintf("Error %d", code)
	}

	if err := errorTmpl.Execute(w, data); err != nil {
		http.Error(w, msg, http.StatusInternalServerError)
	}
}
