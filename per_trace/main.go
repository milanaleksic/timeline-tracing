package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	timeline "github.com/milanaleksic/timeline-from-csv"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	TimeOnlyMs = "15:04:05.999"
)

func main() {

	csvFile := flag.String("csv", "", "input CSV file")
	fieldId := flag.String("fieldId", "", "which field will be used as ID")
	fieldTs := flag.String("fieldTs", "", "which field will be used as timestamp")
	tsFormat := flag.String("tsFormat", "", "how to parse the ts field - use Golang syntax: https://golang.org/pkg/time/#Parse")
	fieldMessage := flag.String("fieldMsg", "", "which field will be used as message")
	beginRegex := flag.String("beginRegex", "", "regex that should have a match on beginning message")
	endRegex := flag.String("endRegex", "", "regex that should have a match on ending message")
	threshold := flag.String("threshold", "1s", "what event length is minimal to consider it")
	// optional
	templateFile := flag.String("templateFile", "template.html", "which Go template file should be used to generate output, use Golang syntax: https://golang.org/pkg/time/#ParseDuration")
	outFile := flag.String("outFile", "output.html", "Where should the output timeline diagram be placed")
	onlyExtremeCase := flag.Bool("onlyExtreme", true, "Expose only extreme case (when most ongoing traces, ignores threshold!)")
	operationRegex := flag.String("operationRegex", "", "regex that should extract (as the first group) the operation name")
	flag.Parse()

	file, err := os.Open(*csvFile)
	if err != nil {
		log.Fatalf("Failed to read the file %v: err=%v", csvFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	allRows, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read as CSV the file %v: err=%v", csvFile, err)
	}

	beginRegexMachine := regexp.MustCompile(*beginRegex)
	endRegexMachine := regexp.MustCompile(*endRegex)
	var operationRegexMachine *regexp.Regexp
	if *operationRegex != "" {
		operationRegexMachine = regexp.MustCompile(*operationRegex)
	}
	thresholdDuration, err := time.ParseDuration(*threshold)
	if err != nil {
		log.Fatalf("Illegal threshold provided, err=%v", err)
	}

	var header = makeHeader(allRows[0], fieldId, fieldTs, fieldMessage)
	var data = allRows[1:]

	var events = make(map[string]Event)
	var ongoing = make(map[string]bool)
	var maxOngoing = make(map[string]bool)

	sort.Slice(data, func(i, j int) bool {
		tsIParsed := parseTs(i, data[i], header, fieldTs, tsFormat)
		tsJParsed := parseTs(j, data[j], header, fieldTs, tsFormat)
		return tsIParsed.Before(tsJParsed)
	})

	for i, x := range data {
		ts := parseTs(i, x, header, fieldTs, tsFormat)
		id := strings.ReplaceAll(x[header[*fieldId]], "\"", "")
		msg := x[header[*fieldMessage]]

		if id == "" {
			continue
		}

		e, ok := events[id]

		if !ok {
			e = Event{
				ID:     id,
				Slices: make([]Slice, 0),
			}
		}

		if beginRegexMachine.FindString(msg) != "" {
			ongoing[e.ID] = true
			if len(ongoing) > len(maxOngoing) {
				maxOngoing = make(map[string]bool)
				for key, value := range ongoing {
					maxOngoing[key] = value
				}
			}
			operation := ""
			if operationRegexMachine != nil {
				matches := operationRegexMachine.FindStringSubmatch(msg)
				if len(matches) != 2 {
					log.Warnf("Unexpected matches in string %v: %+v", msg, matches)
				} else {
					operation = matches[1]
				}
			}
			e.Slices = append(e.Slices, Slice{
				Begin:     ts,
				Operation: operation,
			})
		} else if endRegexMachine.FindString(msg) != "" {
			delete(ongoing, e.ID)
			if len(e.Slices) == 0 {
				log.Warnf("Event without slices encountered for ID %v", e.ID)
			} else {
				e.Slices[len(e.Slices)-1].End = ts
			}
		}

		events[id] = e
	}

	// dump the extreme moment in time
	log.Printf("Max ongoing count of operations is: %d, listing traces:", len(maxOngoing))
	for key := range maxOngoing {
		log.Printf("\t%s", key)
	}

	if *onlyExtremeCase {
		renderTemplateOnlyExtreme(templateFile, events, maxOngoing, *outFile)
	} else {
		renderTemplateWithThreshold(templateFile, events, thresholdDuration, *outFile)
	}
}

func parseTs(rowIndex int, row []string, header map[string]int, fieldTs *string, tsFormat *string) time.Time {
	tsI := row[header[*fieldTs]]
	tsIParsed, err := time.Parse(*tsFormat, tsI)
	if err != nil {
		log.Fatalf("Failed to parse timestamp rowNumber=%v, row=%v, err=%v", rowIndex+1, tsI, err)
	}
	return tsIParsed
}

func renderTemplateOnlyExtreme(templateFile *string, events map[string]Event, maxOngoing map[string]bool, file string) {
	eventsToRender := make(map[string]timeline.EventView)
	for traceID, event := range events {
		slices := make([]timeline.SliceView, 0)
		for _, slice := range event.Slices {
			if _, ok := maxOngoing[traceID]; !ok {
				continue
			}
			if slice.Begin.IsZero() || slice.End.IsZero() {
				continue
			}
			slices = append(slices, timeline.SliceView{
				Operation: slice.Operation,
				Tooltip: fmt.Sprintf("<b>Duration</b>: %d.%d sec<br /><b>Time</b>: %s ... %s",
					slice.End.Sub(slice.Begin).Milliseconds()/1000, slice.End.Sub(slice.Begin).Milliseconds()%1000,
					slice.Begin.Format(TimeOnlyMs), slice.End.Format(TimeOnlyMs)),
				Begin: slice.Begin.UnixNano() / 1000 / 1000,
				End:   slice.End.UnixNano() / 1000 / 1000,
			})
		}
		if len(slices) > 0 {
			eventsToRender[traceID] = timeline.EventView{
				ID:     traceID,
				Slices: slices,
			}
		}
	}
	timeline.RenderTemplateData(templateFile, eventsToRender, file)
}

func renderTemplateWithThreshold(templateFile *string, events map[string]Event, threshold time.Duration, file string) {
	eventsToRender := make(map[string]timeline.EventView)
	for traceID, event := range events {
		slices := make([]timeline.SliceView, 0)
		for _, slice := range event.Slices {
			if slice.Begin.IsZero() || slice.End.IsZero() {
				continue
			}
			if slice.End.Sub(slice.Begin) < threshold {
				continue
			}
			slices = append(slices, timeline.SliceView{
				Operation: slice.Operation,
				Tooltip: fmt.Sprintf("<b>Duration</b>: %d.%d sec<br /><b>Time</b>: %s ... %s",
					slice.End.Sub(slice.Begin).Milliseconds()/1000, slice.End.Sub(slice.Begin).Milliseconds()%1000,
					slice.Begin.Format(TimeOnlyMs), slice.End.Format(TimeOnlyMs)),
				Begin: slice.Begin.UnixNano() / 1000 / 1000,
				End:   slice.End.UnixNano() / 1000 / 1000,
			})
		}
		if len(slices) > 0 {
			eventsToRender[traceID] = timeline.EventView{
				ID:     traceID,
				Slices: slices,
			}
		}
	}
	timeline.RenderTemplateData(templateFile, eventsToRender, file)
}

type Event struct {
	ID     string
	Slices []Slice
}

type Slice struct {
	Operation string
	Begin     time.Time
	End       time.Time
}

func makeHeader(x []string, fieldId *string, fieldTs *string, fieldMessage *string) (header map[string]int) {
	header = make(map[string]int)
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
	return header
}
