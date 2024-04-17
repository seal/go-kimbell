package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"sort"
)

type MetricPair struct {
	Key   string
	Value int
}

type ByValue []MetricPair

func (a ByValue) Len() int           { return len(a) }
func (a ByValue) Less(i, j int) bool { return a[i].Value > a[j].Value }
func (a ByValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("./json/counters.json")
	if err != nil {
		slog.Error(fmt.Sprintf("error opening counters.json %v", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	var metrics map[string]int
	err = json.NewDecoder(file).Decode(&metrics)
	if err != nil {
		slog.Error(fmt.Sprintf("error decoding metrics json %v", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var metricPairs []MetricPair
	for key, value := range metrics {
		metricPairs = append(metricPairs, struct {
			Key   string
			Value int
		}{Key: key, Value: value})
	}
	sort.Sort(ByValue(metricPairs))

	tmpl := template.Must(template.ParseFiles("templates/metrics.html"))
	err = tmpl.Execute(w, metricPairs)
	if err != nil {
		slog.Error(fmt.Sprintf("error executing metrics template %v", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
