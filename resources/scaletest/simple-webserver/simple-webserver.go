package main

import (
    "bytes"
    "flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
    "text/template"
    "time"
	"net/http/httputil"
)

var port string
var tmpl string
var responseTmpl *template.Template

const DEFAULT_TEMPLATE = `{
	"name": "simple-webserver",
	"date": "{{.Date}}",
	"response": "OK"
}`

func main() {
	flag.StringVar(&port, "port", "9090", "The port to serve on")
	flag.StringVar(&tmpl, "template", "", "The JSON response template")
	flag.Parse()

	if tmpl == "" {
		tmpl = os.Getenv("TEMPLATE")
		if tmpl == "" {
			tmpl = DEFAULT_TEMPLATE
		}
	}

	t, err := template.New("response-template").Parse(tmpl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse template:\n %s", err)
		os.Exit(1)
	}
	responseTmpl = t

	bindPort := fmt.Sprintf(":%s", port)
	log.Printf("Started simple web server on %s.", bindPort)
	http.HandleFunc("/", handler)
	http.ListenAndServe(bindPort, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("Request:")
	dump, err := httputil.DumpRequest(r, true)
	if err == nil {
		log.Println(string(dump))
	}
	data := make(map[string]string)
	data["Date"] = time.Now().Format(time.RFC822)

	for _, env := range os.Environ() {
		splits := strings.Split(env, "=")
		if len(splits[0]) > 0 {
			data[splits[0]] = splits[1]
		}
	}

	var response bytes.Buffer
	responseTmpl.Execute(&response, data)

	resp := response.String()
	log.Println("Response:")
	log.Printf("%s", resp)
	fmt.Fprint(w, resp)
}
