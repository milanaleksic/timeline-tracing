package timelineTracing

import (
	"fmt"
	"math"
	"os"
)

func RenderHTMLTemplateData(eventsToRender map[string]EventView, outputFilePath string) error {
	return renderTemplateData(templateHtml, eventsToRender, outputFilePath)
}

func RenderHTMLDatadogTemplateData(eventsToRender map[string]EventView, outputFilePath string) error {
	return renderTemplateData(templateDatadogHtml, eventsToRender, outputFilePath)
}

func renderTemplateData(templateDatadogHtml templateName, eventsToRender map[string]EventView, outputFilePath string) error {
	var minimalTs int64 = math.MaxInt64
	for _, event := range eventsToRender {
		for _, slice := range event.Slices {
			begin := slice.Begin
			if begin < minimalTs {
				minimalTs = begin
			}
		}
	}

	data := TemplateData{
		Events: eventsToRender,
		// let's add 60 sec more for some buffer
		MinimalTs: minimalTs - 60000,
	}

	t, err := loadTemplate(templateDatadogHtml)
	if err != nil {
		return err
	}

	if outputFilePath == "" {
		err := t.Execute(os.Stdout, data)
		if err != nil {
			return fmt.Errorf("failed to fill the template: %w", err)
		}
	} else {
		openFile, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to write to the output file %v: %w", outputFilePath, err)
		}
		defer openFile.Close()

		err = t.Execute(openFile, data)
		if err != nil {
			return fmt.Errorf("failed to fill the template: %w", err)
		}
	}
	return nil
}

type EventView struct {
	ID     string
	Slices []SliceView
}

type SliceView struct {
	Operation string
	Tooltip   string
	Begin     int64
	End       int64
}

type TemplateData struct {
	Events    map[string]EventView
	MinimalTs int64
}
