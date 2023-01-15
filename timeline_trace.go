package timelineTracing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
)

func RenderTraceTemplateData(eventsToRender map[string]EventView, outputFilePath string) error {
	eventsOrdered := orderEventsByStartTs(eventsToRender)
	data, err := convertToTraceEvents(eventsOrdered)
	if err != nil {
		return fmt.Errorf("failed to convert to trace events: %w", err)
	}
	err = writeToFile(data, outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to write trace events: %w", err)
	}
	return nil
}

func writeToFile(data TraceFile, outputFilePath string) error {
	marshal, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}
	err = ioutil.WriteFile(outputFilePath, marshal, 0666)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	return nil
}

func convertToTraceEvents(eventsOrdered []EventView) (TraceFile, error) {
	iter := 1
	traceEvents := make([]TraceEvent, 0)
	minimalTs := eventsOrdered[0].Slices[0].Begin
	for _, event := range eventsOrdered {
		if len(event.Slices) > 2 {
			return TraceFile{}, fmt.Errorf("event has multiple slices: %+v", event.Slices)
		}
		traceEvents = append(traceEvents, TraceEvent{
			Name:          event.Slices[0].Operation,
			CategoriesCSV: "",
			EventType:     Begin,
			Timestamp:     int(event.Slices[0].Begin) * 1000,
			Tid:           iter,
			Args: map[string]any{
				"name":        event.Slices[0].Operation,
				"htmlTooltip": event.Slices[0].Tooltip,
				"trace_id":    event.ID,
				"trace_url":   fmt.Sprintf("https://app.datadoghq.com/apm/trace/%s", event.ID),
				"logs_url":    fmt.Sprintf("https://app.datadoghq.com/logs?query=trace_id%%3A%v&from_ts=%v", event.ID, minimalTs),
			},
		}, TraceEvent{
			Name:          event.Slices[0].Operation,
			CategoriesCSV: "",
			EventType:     End,
			Timestamp:     int(event.Slices[0].End) * 1000,
			Tid:           iter,
			Args: map[string]any{
				"name":        event.Slices[0].Operation,
				"htmlTooltip": event.Slices[0].Tooltip,
				"trace_id":    event.ID,
				"trace_url":   fmt.Sprintf("https://app.datadoghq.com/apm/trace/%s", event.ID),
				"logs_url":    fmt.Sprintf("https://app.datadoghq.com/logs?query=trace_id%%3A%v&from_ts=%v", event.ID, minimalTs),
			},
		})
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
