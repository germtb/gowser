package main

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// SegmentType indicates the type of text segment.
type SegmentType int

const (
	SegmentText SegmentType = iota
	SegmentLink
	SegmentBold
	SegmentItalic
	SegmentCode
)

// Segment represents a piece of styled text.
type Segment struct {
	Text     string
	Type     SegmentType
	URL      string    // For links
	Children []Segment // For links: parsed inline content
}

// LineType indicates the type of line.
type LineType int

const (
	LineNormal LineType = iota
	LineHeading1
	LineHeading2
	LineHeading3
	LineBlockquote
	LineListItem
	LineHorizontalRule
	LineEmpty
)

// ParsedLine represents a line of markdown split into segments.
type ParsedLine struct {
	Type     LineType
	Indent   int
	Segments []Segment
}

// ParseMarkdown parses markdown text into lines of segments using goldmark.
func ParseMarkdown(md string) []ParsedLine {
	source := []byte(md)
	parser := goldmark.DefaultParser()
	doc := parser.Parse(text.NewReader(source))

	var lines []ParsedLine
	parseNode(doc, source, &lines, LineNormal, 0)

	// Ensure we have at least one line
	if len(lines) == 0 {
		lines = append(lines, ParsedLine{Type: LineEmpty, Segments: []Segment{{Text: "", Type: SegmentText}}})
	}

	return lines
}

// parseNode walks the AST and extracts lines with segments.
func parseNode(n ast.Node, source []byte, lines *[]ParsedLine, lineType LineType, indent int) {
	switch node := n.(type) {
	case *ast.Document:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			parseNode(child, source, lines, LineNormal, 0)
		}

	case *ast.Heading:
		lt := LineNormal
		switch node.Level {
		case 1:
			lt = LineHeading1
		case 2:
			lt = LineHeading2
		case 3:
			lt = LineHeading3
		}
		segments := extractInlineSegments(node, source)
		*lines = append(*lines, ParsedLine{Type: lt, Segments: segments})

	case *ast.Paragraph:
		segments := extractInlineSegments(node, source)
		if len(segments) > 0 {
			*lines = append(*lines, ParsedLine{Type: lineType, Indent: indent, Segments: segments})
		}

	case *ast.TextBlock:
		segments := extractInlineSegments(node, source)
		if len(segments) > 0 {
			*lines = append(*lines, ParsedLine{Type: lineType, Indent: indent, Segments: segments})
		}

	case *ast.Blockquote:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			parseNode(child, source, lines, LineBlockquote, 2)
		}

	case *ast.List:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			parseNode(child, source, lines, lineType, indent)
		}

	case *ast.ListItem:
		// Get the list item content
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if para, ok := child.(*ast.Paragraph); ok {
				segments := extractInlineSegments(para, source)
				if len(segments) > 0 {
					// Add bullet point prefix
					bulletSeg := Segment{Text: "• ", Type: SegmentText}
					segments = append([]Segment{bulletSeg}, segments...)
					*lines = append(*lines, ParsedLine{Type: LineListItem, Indent: 2, Segments: segments})
				}
			} else if tb, ok := child.(*ast.TextBlock); ok {
				segments := extractInlineSegments(tb, source)
				if len(segments) > 0 {
					bulletSeg := Segment{Text: "• ", Type: SegmentText}
					segments = append([]Segment{bulletSeg}, segments...)
					*lines = append(*lines, ParsedLine{Type: LineListItem, Indent: 2, Segments: segments})
				}
			} else {
				parseNode(child, source, lines, LineListItem, 2)
			}
		}

	case *ast.ThematicBreak:
		*lines = append(*lines, ParsedLine{
			Type:     LineHorizontalRule,
			Segments: []Segment{{Text: "─────────────────────────────────────────", Type: SegmentText}},
		})

	case *ast.CodeBlock, *ast.FencedCodeBlock:
		// Extract code content
		var buf bytes.Buffer
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			buf.Write(line.Value(source))
		}
		code := strings.TrimSuffix(buf.String(), "\n")
		for _, codeLine := range strings.Split(code, "\n") {
			*lines = append(*lines, ParsedLine{
				Type:     LineNormal,
				Segments: []Segment{{Text: codeLine, Type: SegmentCode}},
			})
		}

	default:
		// For other block nodes, recurse into children
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			parseNode(child, source, lines, lineType, indent)
		}
	}
}

// extractInlineSegments extracts inline segments from a node's children.
func extractInlineSegments(n ast.Node, source []byte) []Segment {
	var segments []Segment
	extractInline(n, source, &segments, SegmentText, "")

	if len(segments) == 0 {
		return []Segment{{Text: "", Type: SegmentText}}
	}
	return segments
}

// extractInline recursively extracts inline elements.
func extractInline(n ast.Node, source []byte, segments *[]Segment, parentType SegmentType, linkURL string) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Text:
			txt := string(node.Segment.Value(source))
			if txt != "" {
				*segments = append(*segments, Segment{Text: txt, Type: parentType})
			}
			// Handle soft/hard line breaks
			if node.SoftLineBreak() || node.HardLineBreak() {
				*segments = append(*segments, Segment{Text: " ", Type: SegmentText})
			}

		case *ast.String:
			txt := string(node.Value)
			if txt != "" {
				*segments = append(*segments, Segment{Text: txt, Type: parentType})
			}

		case *ast.Emphasis:
			// Level 1 = italic (*), Level 2 = bold (**)
			childType := SegmentItalic
			if node.Level == 2 {
				childType = SegmentBold
			}
			extractInline(node, source, segments, childType, linkURL)

		case *ast.CodeSpan:
			txt := string(node.Text(source))
			*segments = append(*segments, Segment{Text: txt, Type: SegmentCode})

		case *ast.Link:
			url := string(node.Destination)
			// Extract children of the link
			var children []Segment
			extractInline(node, source, &children, SegmentText, url)

			// Get display text from children
			var displayText strings.Builder
			for _, ch := range children {
				displayText.WriteString(ch.Text)
			}

			*segments = append(*segments, Segment{
				Text:     displayText.String(),
				Type:     SegmentLink,
				URL:      url,
				Children: children,
			})

		case *ast.AutoLink:
			url := string(node.URL(source))
			*segments = append(*segments, Segment{
				Text:     url,
				Type:     SegmentLink,
				URL:      url,
				Children: []Segment{{Text: url, Type: SegmentText}},
			})

		case *ast.Image:
			// Skip images, or optionally show alt text
			// For now, just skip

		case *ast.RawHTML:
			// Skip raw HTML

		default:
			// Recurse for other inline nodes
			extractInline(child, source, segments, parentType, linkURL)
		}
	}
}
