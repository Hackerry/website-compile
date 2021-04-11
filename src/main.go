package main

import (
	"bufio"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var ComponentPath = filepath.Join("..", "components")
var PagePath = filepath.Join("..", "pages")
var StyleFolderName = "styles"
var StylePath = filepath.Join("..", StyleFolderName)
var BlogFolderName = "blog"
var BlogPath = filepath.Join("..", "pages", "blog")

var StaticPath = filepath.Join("..", "static")
var OutputPath = filepath.Join("..", "output")
var OutputBlogPath = filepath.Join(OutputPath, "blog")
var OutputIndexPagePath = filepath.Join(OutputPath, "index.html")
var OutputBlogIndexPagePath = filepath.Join(OutputBlogPath, "index.html")

type Post struct {
	Tags     string
	FileName string
	Title    string
	Date     string
	Overview string
}

type IndexData struct {
	Projects   []CardDiv
	Tools      []CardDiv
	StyleFiles []string
	Posts      []CardDiv
	RootPath   string
}

type BlogPostData struct {
	StyleFiles []string
	RootPath   string
}

type BlogIndexData struct {
	Posts      []Post
	StyleFiles []string
	RootPath   string
}

type byDate []Post

func (d byDate) Len() int {
	return len(d)
}
func (d byDate) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d byDate) Less(i, j int) bool {
	layout := "Jan. 2, 2006"
	iT, err := time.Parse(layout, d[i].Date)
	Err(err)
	jT, err := time.Parse(layout, d[j].Date)
	Err(err)

	return iT.After(jT)
}

type CardDiv struct {
	Link           string
	Title          string
	Text           string
	AdditionalText string
}

var ProjectDivs = []CardDiv{
	{
		Link:           "https://github.com/Hackerry/Vocabulary_Tracker",
		Title:          "Vocabulary Tracker",
		Text:           "A language learner's vocabulary notebook. Search, manage and review new vocabularies.",
		AdditionalText: "Read More...",
	},
	{
		Link:           "https://github.com/Hackerry/Simulators",
		Title:          "Graph Search Algo Simulator",
		Text:           "Visualize how 5 different graph search algorithms (Greedy, DFS, BFS, Dijkstra, A*) perform on finding the shortest path between two points in an edge weighted graph.",
		AdditionalText: "Read More...",
	},
}

var ToolsDivs = []CardDiv{
	{
		Link:           "/blog/Write A Simple Syntax Highlighter.html",
		Title:          "Code Highlighter",
		Text:           "A simple all-purpose syntax highlighter. Hightlight common reserved keywords and other parameters.",
		AdditionalText: "Explore",
	},
	{
		Link:           "/blog/Make A 3D-Model Wireframe Viewer.html",
		Title:          ".OBJ 3D Model Wireframe Viewer",
		Text:           "A toy project that draws 3D models from .OBJ files.",
		AdditionalText: "Explore",
	},
}

// Helper function that
func copyFiles(sourceDir, destDir string) {
	// Get file infos
	err := filepath.Walk(sourceDir, func(inPath string, inFile fs.FileInfo, err error) error {
		if err != nil {
			Err(err)
		}

		// Copy file to output
		var outPath = filepath.Join(destDir, strings.TrimPrefix(inPath, sourceDir))
		// log.Printf("Copy File: %s\n", outPath)

		sourceFile, err := os.Stat(inPath)
		Err(err)

		// Make dir
		if sourceFile.Mode().IsDir() {
			os.Mkdir(outPath, inFile.Mode().Perm())
		} else {
			inFileStream, err := os.Open(inPath)
			Err(err)
			outFileStream, err := os.Create(outPath)
			Err(err)
			nBytes, err := io.Copy(outFileStream, inFileStream)
			if nBytes != inFile.Size() {
				log.Printf("Copy may be incomplete: %s(%d)->%s(%d)\n", inPath, inFile.Size(), outPath, nBytes)
			}
			inFileStream.Close()
			outFileStream.Close()
		}

		return nil
	})
	if err != nil {
		Err(err)
	}
}

func generateIndexPage(posts []Post) {
	var tmpl *template.Template

	outFile, err := os.Create(OutputIndexPagePath)
	Err(err)

	w := bufio.NewWriter(outFile)
	defer w.Flush()

	// Construct data
	var maxPostShown = 5
	var postsDiv = make([]CardDiv, maxPostShown, maxPostShown)
	for i := 0; i < maxPostShown; i++ {
		postsDiv[i] = CardDiv{
			Link:           filepath.ToSlash(filepath.Join(BlogFolderName, posts[i].FileName)),
			Title:          posts[i].Title,
			Text:           posts[i].Overview,
			AdditionalText: "",
		}
	}
	var indexData = IndexData{
		Projects:   ProjectDivs,
		Tools:      ToolsDivs,
		StyleFiles: []string{filepath.ToSlash(filepath.Join(StyleFolderName, "toplevel.css")), filepath.ToSlash(filepath.Join(StyleFolderName, "index.css")), filepath.ToSlash(filepath.Join(StyleFolderName, "footerHeader.css")), filepath.ToSlash(filepath.Join(StyleFolderName, "cardDiv.css"))},
		Posts:      postsDiv,
		RootPath:   ".",
	}

	// Html tag
	w.WriteString("<html>")

	// Get <head> tag
	tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "head.html")))
	tmpl.Execute(w, indexData)

	// Body tag
	w.WriteString("<body>")

	// Nav-bar section
	w.WriteString("\n\n<!-- Auto-generated navbar -->\n")
	tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "navbar.html")))
	tmpl.Execute(w, indexData)

	// Copied index section
	w.WriteString("\n\n<!-- Copied body -->\n")
	tmpl = template.Must(template.ParseFiles(filepath.Join(PagePath, "index.html"), filepath.Join(ComponentPath, "cardDiv.html")))
	tmpl.Execute(w, indexData)

	w.WriteString("\n\n<!-- Auto-generated footer -->\n")
	tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "footer.html")))
	tmpl.Execute(w, nil)

	// Body & html end tag
	w.WriteString("</body>")
	w.WriteString("</html>")
}

func generateBlogPages() []Post {
	posts := make([]Post, 0, 0)
	titleExp := regexp.MustCompile("<h1>(.*)</h1>")
	dateExp := regexp.MustCompile("<p id='content-date'>(.*)</p>")
	overviewTagExp := regexp.MustCompile("<!--(.*)-->")
	overviewLength := 100
	ellipsis := "..."

	// List all blog files
	err := filepath.Walk(BlogPath, func(inPath string, inFile fs.FileInfo, err error) error {
		if err != nil {
			Err(err)
		}

		// Skip index page and folder
		if (inFile.Mode().IsRegular() && inFile.Name() == "index.html") || inFile.Mode().IsDir() {
			return nil
		}

		// Get post title, date and overview
		content, err := ioutil.ReadFile(inPath)
		Err(err)
		var title, date, overview, tags string
		if matched := titleExp.FindSubmatch(content); len(matched) == 0 {
			log.Fatalf("Title not found")
		} else {
			title = string(matched[1])
		}
		if matched := dateExp.FindSubmatch(content); len(matched) == 0 {
			log.Fatalf("Date not found")
		} else {
			date = string(matched[1])
			date = date[:strings.LastIndex(date, ",")]
		}
		headers := strings.Split(string(content)[:strings.Index(string(content), "<div id='content-wrapper'>")], "\n")
		if len(headers) < 1 || !overviewTagExp.MatchString(headers[0]) {
			log.Fatalf("Overview not found")
		} else {
			overview = string(overviewTagExp.FindSubmatch([]byte(headers[0]))[1])
		}

		if len(headers) < 2 || !overviewTagExp.MatchString(headers[1]) {
			log.Fatalf("Tag not found")
		} else {
			tags = string(overviewTagExp.FindSubmatch([]byte(headers[1]))[1])
		}

		// Cut overview to of <= overviewLength
		if len(overview) >= overviewLength-len(ellipsis) {
			for index := overviewLength - len(ellipsis); index >= 0; index++ {
				if strings.Contains(" .,!?:;'\"", ""+string(overview[index])) {
					overview = overview[:index]
					overview += "..."
					break
				}
			}
		} else {
			overview += "..."
		}

		// log.Println(inPath + " " + title + " " + date + " " + overview + " " + tags)

		// Store information
		posts = append(posts, Post{
			Tags:     tags,
			FileName: inFile.Name(),
			Title:    title,
			Date:     date,
			Overview: overview,
		})

		// Generate page
		var tmpl *template.Template

		var outPath = filepath.Join(OutputBlogPath, inFile.Name())
		outFile, err := os.Create(outPath)
		Err(err)

		w := bufio.NewWriter(outFile)
		defer w.Flush()

		var blogPostData = BlogPostData{
			StyleFiles: []string{filepath.ToSlash(filepath.Join(StylePath, "toplevel.css")), filepath.ToSlash(filepath.Join(StylePath, "footerHeader.css")), filepath.ToSlash(filepath.Join(StylePath, "post.css"))},
			RootPath:   "..",
		}

		// Html tag
		w.WriteString("<html>")

		// Get <head> tag
		tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "head.html")))
		tmpl.Execute(w, blogPostData)

		// Body tag
		w.WriteString("<body>")

		// Nav-bar section
		w.WriteString("\n\n<!-- Auto-generated navbar -->\n")
		tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "navbar.html")))
		tmpl.Execute(w, blogPostData)

		// Copied post section
		w.WriteString("\n\n<!-- Copied body -->\n")
		tmpl = template.Must(template.ParseFiles(inPath))
		tmpl.Execute(w, blogPostData)

		w.WriteString("\n\n<!-- Auto-generated footer -->\n")
		tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "footer.html")))
		tmpl.Execute(w, nil)

		// Body & html end tag
		w.WriteString("</body>")
		w.WriteString("</html>")

		return nil
	})

	Err(err)
	// log.Println(posts)

	// sort posts by date
	sort.Sort(byDate(posts))

	return posts
}

func generateBlogIndexPage(posts []Post) {
	outFile, err := os.Create(OutputBlogIndexPagePath)
	Err(err)

	w := bufio.NewWriter(outFile)
	defer w.Flush()

	var tmpl *template.Template

	var blogIndexData = BlogIndexData{
		Posts:      posts,
		StyleFiles: []string{filepath.ToSlash(filepath.Join(StylePath, "toplevel.css")), filepath.ToSlash(filepath.Join(StylePath, "footerHeader.css")), filepath.ToSlash(filepath.Join(StylePath, "blogIndex.css"))},
		RootPath:   "..",
	}

	// Html tag
	w.WriteString("<html>")

	// Get <head> tag
	tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "head.html")))
	tmpl.Execute(w, blogIndexData)

	// Body tag
	w.WriteString("<body>")

	// Nav-bar section
	w.WriteString("\n\n<!-- Auto-generated navbar -->\n")
	tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "navbar.html")))
	tmpl.Execute(w, blogIndexData)

	// Copied index section
	w.WriteString("\n\n<!-- Copied body -->\n")
	tmpl = template.Must(template.ParseFiles(filepath.Join(BlogPath, "index.html")))
	tmpl.Execute(w, blogIndexData)

	w.WriteString("\n\n<!-- Auto-generated footer -->\n")
	tmpl = template.Must(template.ParseFiles(filepath.Join(ComponentPath, "footer.html")))
	tmpl.Execute(w, nil)

	// Body & html end tag
	w.WriteString("</body>")
	w.WriteString("</html>")
}

func main() {
	// Delete output folder before moving files
	Err(os.RemoveAll(OutputPath))
	copyFiles(StaticPath, OutputPath)

	// Create blog folder
	Err(os.Mkdir(filepath.Join(OutputPath, "blog"), 0666))

	// Generate blog posts
	posts := generateBlogPages()

	// Generate blog main page
	generateBlogIndexPage(posts)

	// Generate index page
	generateIndexPage(posts)
}

func Err(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
