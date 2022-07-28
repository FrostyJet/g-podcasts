package main

import (
	"crawler/internal/db"
	"crawler/internal/models"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

const filesPath = "./resources/metadata"

func init() {
	db.Init()
}

func main() {
	files := getPodcastsToPersist()

	for _, f := range files {
		if strings.Contains(f.Name(), ".json") {
			continue
		}

		p := createPodcastFromFile(fmt.Sprintf("%s/%s", filesPath, f.Name()))

		podcastID := savePodcast(&p)

		ok := saveTracks(podcastID, p.Tracks)

		fmt.Println(ok)
	}
}

func savePodcast(p *models.Podcast) int {
	store := db.GetDB()

	query := `INSERT INTO podcasts (title, description, poster, date_created) VALUES($1, $2, $3, $4) RETURNING id`

	stmt, err := store.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var insertID int
	err = stmt.QueryRow(p.Title, p.Description, p.Poster, time.Now()).Scan(&insertID)
	if err != nil {
		log.Fatal(err)
	}

	return insertID
}

func saveTracks(podcastID int, tracks []string) bool {
	store := db.GetDB()

	query := "INSERT INTO tracks (podcast_id, path, date_created) VALUES"

	params := make([]interface{}, 0, len(tracks)*3)
	queryTail := ""

	t := time.Now()
	for index, track := range tracks {
		i := index * 3
		queryTail += fmt.Sprintf("($%d,$%v,$%v),", i+1, i+2, i+3)
		params = append(params, podcastID, track, t)
	}

	query += queryTail[:len(queryTail)-1]

	stmt, err := store.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	stmt.Exec(params...)

	return true
}

func createPodcastFromFile(path string) models.Podcast {
	p := models.Podcast{}

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(contents, &p)
	if err != nil {
		log.Fatal(err)
	}

	return p
}

func getPodcastsToPersist() []fs.FileInfo {
	list, err := ioutil.ReadDir(filesPath)
	if err != nil {
		log.Fatal(err)
	}

	return list
}
