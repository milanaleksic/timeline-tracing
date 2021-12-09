package main

import (
	"encoding/csv"
	"flag"
	"html/template"
	"log"
	"os"
	"regexp"
	"time"
)

func main() {

	csvFile := flag.String("csv", "", "input CSV file")
	fieldId := flag.String("fieldId", "", "which field will be used as ID")
	fieldTs := flag.String("fieldTs", "", "which field will be used as timestamp")
	tsFormat := flag.String("tsFormat", "", "how to parse the ts field - use Golang syntax: https://golang.org/pkg/time/#Parse")
	fieldMessage := flag.String("fieldMsg", "", "which field will be used as message")
	beginRegex := flag.String("beginRegex", "", "regex that should have a match on beginning message, use (...) to match an identifier")
	endRegex := flag.String("endRegex", "", "regex that should have a match on ending message, use (...) to match an identifier")
	threshold := flag.String("threshold", "1s", "what event length is minimal to consider it")
	// optional
	templateFile := flag.String("templateFile", "template.html", "which Go template file should be used to generate output, use Golang syntax: https://golang.org/pkg/time/#ParseDuration")
	outFile := flag.String("outFile", "output.html", "Where should the output timeline diagram be placed")
	flag.Parse()

	file, err := os.Open(*csvFile)
	if err != nil {
		log.Fatalf("Failed to read the file %v: err=%v", csvFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	all, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read as CSV the file %v: err=%v", csvFile, err)
	}

	beginRegexMachine := regexp.MustCompile(*beginRegex)
	endRegexMachine := regexp.MustCompile(*endRegex)
	thresholdDuration, err := time.ParseDuration(*threshold)
	if err != nil {
		log.Fatalf("Illegal threshold provided, err=%v", err)
	}

	var header = make(map[string]int)
	var events = make(map[string]Event)
	for i, x := range all {
		if i == 0 {
			makeHeader(x, header, fieldId, fieldTs, fieldMessage)
			continue
		}
		ts, err := time.Parse(*tsFormat, x[header[*fieldTs]])
		if err != nil {
			log.Fatalf("Failed to parse timestamp rowNumber=%v, row=%v, err=%v", i+1, x, err)
		}
		msg := x[header[*fieldMessage]]

		id := x[header[*fieldId]]
		matchesBegin := beginRegexMachine.FindAllStringSubmatch(msg, -1)
		matchesEnd := endRegexMachine.FindAllStringSubmatch(msg, -1)
		if len(matchesBegin) != 0 {
			eventID := matchesBegin[0][1] + "_" + id
			e := getOrMakeRecord(events, eventID)
			e.Begin = ts
			events[e.ID] = e
		} else if len(matchesEnd) != 0 {
			eventID := matchesEnd[0][1] + "_" + id
			e := getOrMakeRecord(events, eventID)
			e.End = ts
			events[e.ID] = e
		}
	}

	renderTemplate(templateFile, events, thresholdDuration, *outFile)
}

func getOrMakeRecord(events map[string]Event, txId string) Event {
	e, ok := events[txId]
	if !ok {
		e = Event{
			ID: txId,
		}
	}
	events[txId] = e
	return e
}

func renderTemplate(templateFile *string, events map[string]Event, threshold time.Duration, file string) {

	eventsToRender := make(map[string]EventView)

	for traceID, event := range events {
		if event.Begin.IsZero() || event.End.IsZero() {
			continue
		}
		if event.End.Sub(event.Begin) < threshold {
			continue
		}
		eventsToRender[traceID] = EventView{
			ID:    traceID,
			Begin: event.Begin.UnixNano() / 1000 / 1000,
			End:   event.End.UnixNano() / 1000 / 1000,
		}
	}

	templateTimeline := template.New("timeline")
	t, err := templateTimeline.ParseFiles(*templateFile)
	if err != nil {
		log.Fatalf("Failed to parse the template file %v: err=%v", *templateFile, err)
	}

	if file == "" {
		err = t.Execute(os.Stdout, eventsToRender)
		if err != nil {
			log.Fatalf("Failed to fill the template err=%v", err)
		}
	} else {
		openFile, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Fatalf("Failed to write to the output file %v err=%v", file, err)
		}
		err = t.Execute(openFile, eventsToRender)
		if err != nil {
			log.Fatalf("Failed to fill the template err=%v", err)
		}
	}
}

type Event struct {
	ID    string
	Begin time.Time
	End   time.Time
}

type EventView struct {
	ID    string
	Begin int64
	End   int64
}

func makeHeader(x []string, header map[string]int, fieldId *string, fieldTs *string, fieldMessage *string) {
	for j, h := range x {
		header[h] = j
	}

	var ok bool
	_, ok = header[*fieldId]
	if !ok {
		log.Fatalf("Id Field %v not found in header %v", *fieldId, header)
	}
	_, ok = header[*fieldTs]
	if !ok {
		log.Fatalf("Ts Field %v not found in header %v", *fieldTs, header)
	}
	_, ok = header[*fieldMessage]
	if !ok {
		log.Fatalf("Message Field %v not found in header %v", *fieldMessage, header)
	}
}
