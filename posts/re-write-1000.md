## A constant re-write ... 
After little activity on my re-write of this blog I noticed something that hadn't occurred to me previously.

In the process of over-engineering a simple blog, using new technologies to me ( Vue.js ) and an unfamiliar framework in rust ( Tide) I had removed my *want* to update this site.

Code that had left my mind, and a framework that hurt my brain...

The obvious solution to me ? Yet *another* re-write, but with the aim for it to be my last.

## The Minimalist Approach

I've tried JS frameworks, Angular, React & Vue.js to be specific, I always find myself with hours and hours of work with a finished problem but a weird feeling.

The feeling that I didn't accomplish anything.

After some thought and a discussion with a friend it hit me, I hated 

	 Glue engineering

The frameworks and work I had done didn't feel like building, it felt like gluing a large majority of technologies I didn't write, nor do I understand into a mammoth of a dist file... 

With this in mind, I decided my next front-end en devour would be minimalist, much like the original version of this site.

By adopting a minimalist approach, I aimed to:
- Reduce complexity and maintain a clean codebase
- Improve performance by eliminating unnecessary overhead
- Enhance maintainability and ease of understanding
- Focus on the essential features of a blog

## The Technology Stack

To align with this new founded idea, I chose the below stack for various reasons:

- **Go**: Large familiarity, *speed* & low complexity.
- **Gorilla Mux**: Established, mature, and I have not yet tried [https://tip.golang.org/doc/go1.22](https://tip.golang.org/doc/go1.22)
- **Markdown**: All my previous posts have been written in markdown, but the goal this time is for the ability to *edit without editing a database*. For quick, smaller changes
- **gomarkdown**: A Go library for converting Markdown to HTML, gomarkdown simplifies the rendering process.

## The Architecture

The architecture of the blog service follows a straightforward and minimalist design:

1. **HTTP Handlers**: The endpoints should be simple ( there's only 5 - technically 4 ) which are:
	1. r.HandleFunc("/api/new", handlers.NewPost).Methods("POST")
	2. r.HandleFunc("/", handlers.IndexHandler)
	3. r.HandleFunc("/posts/{url}", handlers.GetPost).Methods("GET")
	4. r.HandleFunc("/index.html", handlers.IndexHandler) 
	5. r.HandleFunc("/tomorrow-night.css", ........)

2. **Post Storage**: Posts will be stored as .md files with a corresponding .json file, with the added benefit of being re-generated upon each start of the server.

3. **Templates**: Go's text/template is a no-brainer, a simple yet mature templating library, as I do not need extra features. 

4. **Slog**: I've used zap, log, etc etc and need something not as *fast and engineered* as Zap by Uber, but with better control & debug levels, for this I settled on Slog.


## The code 

#### Logging 
Keeping it simple, my logging configuration is as follows:
```go
	file, err := os.OpenFile("log.json", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("Failed to open log file", err)
		os.Exit(1)
	}
	defer file.Close()
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.MultiWriter(file, os.Stdout), &slog.HandlerOptions{
		Level: slog.LevelInfo})))
```

StdOut & file logging in a nice json format, for example:
```json
{"time":"2024-04-13T14:54:37.240131824+01:00","level":"INFO","msg":"Starting server"}

```


#### Index ep:
![https://img.kimbell.uk/gPJQ99.png](https://img.kimbell.uk/gPJQ99.png)

1.  Get Posts returns an array of Post{} with information of the posts normally, but here we are using the time of the most recently modified post.
2. Parse index re-gen's index.html ( Which we only do if a post has been modified)

#### NewPost

NewPost is not too interesting, some json marshalling, validation & writing the .json and .md file 

![https://img.kimbell.uk/gDQ9GB.png](https://img.kimbell.uk/gDQ9GB.png)

...etc etc etc

#### Get Post

Simple yet again, read a file, return it 

![https://img.kimbell.uk/TDEQTC.png](https://img.kimbell.uk/TDEQTC.png)

#### Utils

The most interesting part of this is in our util package, the functions which turn MD to HTML. Using [github.com/gomarkdown/markdown](github.com/gomarkdown/markdown) for our html generation with a couple short functions

##### Generate Posts
```go
func GeneratePosts() error {
	posts, _, err := GetPosts()
	if err != nil {
		return err
	}

	err = os.MkdirAll("generated", os.ModePerm)
	if err != nil {
		return err
	}

	for _, post := range posts {
		markdownFileName := filepath.Join("posts", post.Url+".md")
		markdownData, err := os.ReadFile(markdownFileName)
		if err != nil {
			return err
		}

		htmlContent := mdToHTML(markdownData)

		tem, err := os.ReadFile("templates/post.html")
		if err != nil {
			return err
		}
		tmpl := template.Must(template.New("post").Parse(string(tem)))
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, struct {
			Title   string
			Date    string
			Content template.HTML
		}{
			Title:   post.Title,
			Date:    post.Date,
			Content: template.HTML(htmlContent),
		})
		if err != nil {
			return err
		}

		htmlFileName := filepath.Join("generated", post.Url+".html")
		err = os.WriteFile(htmlFileName, buf.Bytes(), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}
```

##### MD To Html

```go
func mdToHTML(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

```

#### Tests
Just kidding... you're not going to read that 
Although if you wanted to you can find the repo here:
[https://github.com/seal/go-kimbell](https://github.com/seal/go-kimbell)

## Conclusion

All in all this was a fairly rewarding experience leaving a good feeling in my mind.A minimalist front-end ( which I have intentionally left out, you're currently using it) and a simple yet effective go backend leading to what I feel like is a finished product ( minus the possibility of a read counter ? )

I think going forward I'll be adopting this simple mentality more often than not, as I have spent many an hour confused over my *own code* from as little as 2-3 months ago.

