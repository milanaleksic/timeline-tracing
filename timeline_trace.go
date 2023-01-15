package timelineTracing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

func RenderTracePerfettoTemplateData(eventsToRender map[string]EventView, outputFilePath string) error {
	t, err := loadTemplate(templateTraceHtml)
	if err != nil {
		return err
	}

	data, err := getTraceData(eventsToRender)
	if err != nil {
		return err
	}

	marshal, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	templateData := map[string]any{
		"Data": string(marshal),
	}

	if outputFilePath == "" {
		err := t.Execute(os.Stdout, templateData)
		if err != nil {
			return fmt.Errorf("failed to fill the template: %w", err)
		}
	} else {
		openFile, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to write to the output file %v: %w", outputFilePath, err)
		}
		defer openFile.Close()

		err = t.Execute(openFile, templateData)
		if err != nil {
			return fmt.Errorf("failed to fill the template: %w", err)
		}
	}
	return nil
}

func GenerateTraceTemplateData(eventsToRender map[string]EventView, outputFilePath string) error {
	data, err := getTraceData(eventsToRender)
	if err != nil {
		return err
	}

	marshal, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	if outputFilePath == "" {
		_, err = os.Stdout.Write(marshal)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	} else {
		err = ioutil.WriteFile(outputFilePath, marshal, 0666)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	}

	return nil
}

func getTraceData(eventsToRender map[string]EventView) (TraceFile, error) {
	eventsOrdered := orderEventsByStartTs(eventsToRender)
	data, err := convertToTraceEvents(eventsOrdered)
	if err != nil {
		return TraceFile{}, fmt.Errorf("failed to convert to trace events: %w", err)
	}
	return data, nil
}

func convertToTraceEvents(eventsOrdered []EventView) (TraceFile, error) {
	iter := 1
	traceEvents := make([]TraceEvent, 0)
	if len(eventsOrdered) == 0 {
		return TraceFile{}, nil
	}
	minimalTs := eventsOrdered[0].Slices[0].Begin
	for _, event := range eventsOrdered {
		for _, slice := range event.Slices {
			traceEvents = append(traceEvents, TraceEvent{
				Name:          slice.Operation,
				CategoriesCSV: "",
				EventType:     Begin,
				Timestamp:     int(slice.Begin) * 1000,
				Tid:           iter,
				Args: map[string]any{
					"name":        slice.Operation,
					"htmlTooltip": slice.Tooltip,
					"trace_id":    event.ID,
					"trace_url":   fmt.Sprintf("https://app.datadoghq.com/apm/trace/%s", event.ID),
					"logs_url":    fmt.Sprintf("https://app.datadoghq.com/logs?query=trace_id%%3A%v&from_ts=%v", event.ID, minimalTs),
				},
			}, TraceEvent{
				Name:          slice.Operation,
				CategoriesCSV: "",
				EventType:     End,
				Timestamp:     int(slice.End) * 1000,
				Tid:           iter,
				Args: map[string]any{
					"name":        slice.Operation,
					"htmlTooltip": slice.Tooltip,
					"trace_id":    event.ID,
					"trace_url":   fmt.Sprintf("https://app.datadoghq.com/apm/trace/%s", event.ID),
					"logs_url":    fmt.Sprintf("https://app.datadoghq.com/logs?query=trace_id%%3A%v&from_ts=%v", event.ID, minimalTs),
				},
			})
		}
		iter++
	}
	data := TraceFile{
		TraceEvents:     traceEvents,
		DisplayTimeUnit: "ms",
	}
	return data, nil
}

func orderEventsByStartTs(eventsToRender map[string]EventView) []EventView {
	eventsOrdered := make([]EventView, 0)
	for _, event := range eventsToRender {
		eventsOrdered = append(eventsOrdered, event)
	}
	sort.SliceStable(eventsOrdered, func(i, j int) bool {
		return eventsOrdered[i].Slices[0].Begin < eventsOrdered[j].Slices[0].Begin
	})
	return eventsOrdered
}

type EventType string

const (
	Begin         EventType = "B"
	End           EventType = "E"
	CompleteEvent           = "X"
)

type TraceEvent struct {
	Name          string         `json:"name"`
	CategoriesCSV string         `json:"cat"`
	EventType     EventType      `json:"ph"`
	Timestamp     int            `json:"ts"`
	Pid           int            `json:"pid"`
	Tid           int            `json:"tid"`
	Args          map[string]any `json:"args"`
}

type TraceFile struct {
	TraceEvents     []TraceEvent      `json:"traceEvents"`
	DisplayTimeUnit string            `json:"displayTimeUnit"`
	OtherData       map[string]string `json:"otherData"`
}
