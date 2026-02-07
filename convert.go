package main

import (
	"fmt"
	"net/url"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	readability "github.com/go-shiori/go-readability"
)

// Result contains the converted content and metadata.
type Result struct {
	Title    string
	Markdown string
}

// ConvertToMarkdown converts HTML to Markdown using readability for content extraction.
func ConvertToMarkdown(html string, baseURL string) (Result, error) {
	return ConvertToMarkdownWithOptions(html, baseURL, true)
}

// ConvertToMarkdownRaw converts HTML to Markdown without readability extraction.
func ConvertToMarkdownRaw(html string, baseURL string) (Result, error) {
	return ConvertToMarkdownWithOptions(html, baseURL, false)
}

// ConvertToMarkdownWithOptions converts HTML with optional readability extraction.
func ConvertToMarkdownWithOptions(html string, baseURL string, useReadability bool) (Result, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return Result{}, fmt.Errorf("parsing URL: %w", err)
	}

	if !useReadability {
		return convertRaw(html, baseURL)
	}

	// Use readability to extract main content
	article, err := readability.FromReader(strings.NewReader(html), parsedURL)
	if err != nil || len(strings.TrimSpace(article.Content)) < 100 {
		// Fallback to raw HTML if readability fails or returns too little content
		return convertRaw(html, baseURL)
	}

	// Convert the extracted content to markdown
	markdown, err := htmltomarkdown.ConvertString(
		article.Content,
		converter.WithDomain(baseURL),
	)
	if err != nil {
		return Result{}, fmt.Errorf("converting HTML to markdown: %w", err)
	}

	title := article.Title
	if title == "" {
		title = parsedURL.Host
	}

	return Result{
		Title:    title,
		Markdown: markdown,
	}, nil
}

// convertRaw converts HTML without readability extraction (fallback).
func convertRaw(html string, baseURL string) (Result, error) {
	parsedURL, _ := url.Parse(baseURL)

	markdown, err := htmltomarkdown.ConvertString(
		html,
		converter.WithDomain(baseURL),
	)
	if err != nil {
		return Result{}, fmt.Errorf("converting HTML to markdown: %w", err)
	}

	return Result{
		Title:    parsedURL.Host,
		Markdown: markdown,
	}, nil
}
