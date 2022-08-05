package timelineFromCSV

import (
	"html/template"
	"log"
	"math"
	"os"
)

func RenderTemplateData(templateFile *string, eventsToRender map[string]EventView, outputFilePath string) {
	templateTimeline := template.New("timeline")
	t, err := templateTimeline.ParseFiles(*templateFile)
	if err != nil {
		log.Fatalf("Failed to parse the template file %v: err=%v", *templateFile, err)
	}

	var minimalTs int64 = math.MaxInt64
	for _, event := range eventsToRender {
		begin := event.Begin
		if begin < minimalTs {
			minimalTs = begin
		}
	}

	data := TemplateData{
		Events: eventsToRender,
		// let's add 60 sec more for some buffer
		MinimalTs: minimalTs - 60000,
	}

	if outputFilePath == "" {
		err = t.Execute(os.Stdout, data)
		if err != nil {
			log.Fatalf("Failed to fill the template err=%v", err)
		}
	} else {
		openFile, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("Failed to write to the output file %v err=%v", outputFilePath, err)
		}
		err = t.Execute(openFile, data)
		if err != nil {
			log.Fatalf("Failed to fill the template err=%v", err)
		}
	}
}

type EventView struct {
	ID    string
	Begin int64
	End   int64
}

type TemplateData struct {
	Events    map[string]EventView
	MinimalTs int64
}
