package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"crawler/internal/models"

	"golang.org/x/net/html"
)

type any = interface{}

const url = "https://podcasts.google.com/"
const resourcesPath = "./resources"
const trendingPodcastsPath = resourcesPath + "/html/trendingPodcastsPath.html"

var options map[string]any = map[string]any{
	"maxTracksPerPodcast": 5,
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	CacheTrendingPodcastsPage()

	links := ParseTrendingPodcastLinks()

	fmt.Printf("Found %d podcasts to download\n", len(links))

	var wg sync.WaitGroup

	limit := 20 //len(links)
	done := 0

	wg.Add(limit)
	for index, link := range links {

		go func(index int, link string) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered. Error:\n", r)
				}
			}()

			if !strings.Contains(link, "https://") {
				link = url + link[2:]
			}

			fmt.Println(link)

			filePath := fmt.Sprintf("%s/html/tmp_%d.html", resourcesPath, index)

			CachePage(link, filePath)
			CreatePodcastFromPage(filePath)
			podcastData := CreatePodcastFromPage(filePath)
			podcastData.OriginalUrl = link

			PersistPodcast(podcastData)

			done++
			fmt.Printf("Done %d from %d\n", done, limit)
			wg.Done()
		}(index, link)

		if index >= limit {
			break
		}
	}

	wg.Wait()
	fmt.Println("Finished successfully")
}

func CacheTrendingPodcastsPage() {
	log.Println("Caching trending podcasts page")
	CachePage(url, trendingPodcastsPath)
}

func extractAttr(node *html.Node, targetAttr string) string {
	for _, attr := range node.Attr {
		if attr.Key == targetAttr {
			return attr.Val
		}
	}

	return ""
}

func collectText(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, buf)
	}
}

func PersistPodcast(p models.Podcast) {
	reg, err := regexp.Compile("[^a-zA-Z0-9 ]+")
	if err != nil {
		log.Panic(err)
	}

	fileName := reg.ReplaceAllString(p.Title, "")
	fileName = strings.ReplaceAll(strings.ToLower(fileName), " ", "_")
	path := fmt.Sprintf("%s/metadata/%s.json", resourcesPath, fileName)

	_, err = os.Open(path)
	if !os.IsNotExist(err) {
		fmt.Printf("Skipping persisting of '%s', already exists\n", p.Title)
		return
	}

	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("Could not persist podcast to destination: %s\n", path)
		log.Panic(err)
	}
	defer file.Close()

	contents, err := json.Marshal(p)
	if err != nil {
		fmt.Printf("Could not create json object from podcast: %v\n", p)
		log.Panic(err)
	}

	n, err := file.Write(contents)
	if err != nil {
		fmt.Printf("Could not persist json contents of podcast to destination: %s\n", path)
		log.Panic(err)
	}

	fmt.Printf("Successfully persisted podcast: %s, with %d bytes\n", p.Title, n)
}

func CreatePodcastFromPage(filePath string) models.Podcast {
	p := models.Podcast{
		Tracks: make([]string, 0, 5),
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Could not open file: %s\n", filePath)
		log.Panic(err)
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		log.Printf("Could not parse html from file: %s", filePath)
		log.Panic(err)
	}

	titleClassNames := map[string]bool{
		"ZfMIwb": true,
		"ik7nMd": true,
		"wv3SK":  true,
	}

	descriptionTargetAttrs := map[string]bool{
		"YGHahd": true,
		"QpaWg":  true,
		"GHG3g":  true,
	}

	const jsModelTarget = "kY0ub"
	const imgClassName = "BhVIWc"

	var f func(*html.Node)
	f = func(n *html.Node) {

		// Title
		if n.Type == html.ElementNode && (n.Data == "a" || n.Data == "div") {
			className := extractAttr(n, "class")
			if _, ok := titleClassNames[className]; ok {
				text := &bytes.Buffer{}
				collectText(n, text)
				p.Title = text.String()
			}
		}

		// Description
		if n.Type == html.ElementNode && (n.Data == "div") {
			jsName := extractAttr(n, "jsname")
			if _, ok := descriptionTargetAttrs[jsName]; ok {
				text := &bytes.Buffer{}
				collectText(n, text)
				p.Description = text.String()
			}
		}

		// Track (Sound)
		if len(p.Tracks) < options["maxTracksPerPodcast"].(int) {
			if n.Type == html.ElementNode && n.Data == "div" {
				jsModel := extractAttr(n, "jsmodel")
				if jsModel == jsModelTarget {
					trackUrl := extractAttr(n, "jsdata")
					trackUrl = strings.Replace(trackUrl, "Kwyn5e;", "", 1)

					trackID := DownloadTrack(trackUrl)
					p.Tracks = append(p.Tracks, fmt.Sprintf("%s.mp3", trackID))
				}
			}
		}

		// Poster
		if n.Type == html.ElementNode && n.Data == "img" {
			className := extractAttr(n, "class")
			if className == imgClassName {
				imageSrc := extractAttr(n, "src")

				imageID := DownloadImage(imageSrc)
				p.Poster = imageID + ".jpeg"
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return p
}

func DownloadImage(url string) string {

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Could not download image from URL: %s\n", url)
		log.Panic(err)
	}
	defer resp.Body.Close()

	hash := md5Encode(url)
	filePath := fmt.Sprintf("%s/images/%s.jpeg", resourcesPath, hash)

	_, err = os.Open(filePath)
	if !os.IsNotExist(err) {
		fmt.Printf("Skipping persisting of Poster '%s', already exists\n", hash)
		return hash
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Could not create file to store image: %s\n", filePath)
		log.Panic(err)
	}
	defer file.Close()

	n, err := io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Could not download image from URL: %s\n", url)
		log.Panic(err)
	}

	fmt.Printf("Successfully stored poster image with contents of %v bytes\n", n)

	return hash
}

func md5Encode(str string) string {
	h := md5.New()
	io.WriteString(h, str)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func DownloadTrack(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Could not download contents from URL: %s\n", url)
		log.Panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Could not download contents from URL: %s\n", url)
		log.Panic(err)
	}

	hash := md5Encode(url)
	filePath := fmt.Sprintf("%s/tracks/%s.mp3", resourcesPath, hash)

	_, err = os.Open(filePath)
	if !os.IsNotExist(err) {
		fmt.Printf("Skipping persisting of Audio track '%s', already exists\n", hash)
		return hash
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Could not create a file: %s\n", filePath)
		log.Panic(err)
	}
	defer file.Close()

	n, err := io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Could not write contents of audio into file: %s\n", url)
		log.Panic(err)
	}

	fmt.Printf("Successfully stored audio file with contents of %v bytes\n", n)
	return hash
}

func ParseTrendingPodcastLinks() []string {
	file, err := os.Open(trendingPodcastsPath)
	if err != nil {
		log.Println("Could not open temporary file to parse")
		log.Panic(err)
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		log.Println("Could not parse html contents")
		log.Panic(err)
	}

	links := make([]string, 0, 20)

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := extractAttr(n, "href")
			if len(href) > 5 && strings.Contains(href, "/feed/") {
				links = append(links, href)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return links
}

func CachePage(url, filePath string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Could not make GET request to URL %v\n", url)
		log.Panic(err)
	}

	defer resp.Body.Close()
	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Could not read contents of response")
		log.Panic(err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Could not create temporary file to write response")
		log.Panic(err)
	}

	defer file.Close()
	n, err := file.WriteString(string(contents))
	if err != nil {
		fmt.Println("Coudld not write contents into temporary created file")
		log.Panic(err)
	}

	fmt.Printf("Successfully written contents into temporary file, %s %v bytes\n", filePath, n)
}
