package service

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

const maxCreativeSVGBytes = 1 << 20

var creativeSVGElements = map[string]struct{}{
	"svg": {}, "g": {}, "path": {}, "rect": {}, "circle": {}, "ellipse": {}, "line": {}, "polyline": {}, "polygon": {}, "text": {}, "tspan": {}, "defs": {}, "linearGradient": {}, "radialGradient": {}, "stop": {}, "clipPath": {}, "mask": {}, "filter": {}, "feGaussianBlur": {}, "title": {}, "desc": {},
}

var creativeSVGAttributes = map[string]struct{}{
	"xmlns": {}, "xmlns:xlink": {}, "viewBox": {}, "width": {}, "height": {}, "x": {}, "y": {}, "x1": {}, "x2": {}, "y1": {}, "y2": {}, "cx": {}, "cy": {}, "r": {}, "rx": {}, "ry": {}, "d": {}, "points": {}, "fill": {}, "fill-opacity": {}, "fill-rule": {}, "stroke": {}, "stroke-width": {}, "stroke-linecap": {}, "stroke-linejoin": {}, "stroke-opacity": {}, "opacity": {}, "transform": {}, "font-family": {}, "font-size": {}, "font-weight": {}, "text-anchor": {}, "offset": {}, "stop-color": {}, "stop-opacity": {}, "id": {}, "clip-path": {}, "mask": {}, "filter": {}, "stdDeviation": {},
}

func SanitizeCreativeSVG(raw []byte, maxBytes int64) ([]byte, error) {
	text := strings.TrimSpace(string(raw))
	if strings.HasPrefix(text, "```") {
		if lineEnd := strings.IndexByte(text, '\n'); lineEnd > 3 && strings.EqualFold(strings.TrimSpace(text[3:lineEnd]), "svg") && strings.HasSuffix(text, "```") {
			text = strings.TrimSpace(text[lineEnd+1 : len(text)-3])
		}
	}
	raw = []byte(text)
	if maxBytes <= 0 || maxBytes > maxCreativeSVGBytes {
		maxBytes = maxCreativeSVGBytes
	}
	if len(raw) == 0 || int64(len(raw)) > maxBytes {
		return nil, errors.New("SVG exceeds the allowed size")
	}
	lower := strings.ToLower(string(raw))
	if strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<!entity") || strings.Contains(lower, "<script") || strings.Contains(lower, "<foreignobject") {
		return nil, errors.New("SVG contains prohibited markup")
	}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	depth := 0
	seenRoot := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.New("invalid SVG XML")
		}
		switch value := token.(type) {
		case xml.StartElement:
			if _, allowed := creativeSVGElements[value.Name.Local]; !allowed {
				return nil, fmt.Errorf("SVG element %q is not allowed", value.Name.Local)
			}
			if !seenRoot {
				if value.Name.Local != "svg" {
					return nil, errors.New("SVG root element is required")
				}
				seenRoot = true
			}
			for _, attribute := range value.Attr {
				name := attribute.Name.Local
				if attribute.Name.Space == "xmlns" && name != "xmlns" {
					name = "xmlns:" + name
				}
				if strings.HasPrefix(name, "xmlns") {
					continue
				}
				_, allowed := creativeSVGAttributes[name]
				if !allowed || strings.HasPrefix(strings.ToLower(name), "on") || unsafeSVGValue(attribute.Value) {
					return nil, fmt.Errorf("SVG attribute %q is not allowed", name)
				}
			}
			depth++
			if err := encoder.EncodeToken(value); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if depth == 0 {
				return nil, errors.New("invalid SVG root")
			}
			depth--
			if err := encoder.EncodeToken(value); err != nil {
				return nil, err
			}
		case xml.CharData:
			if err := encoder.EncodeToken(value); err != nil {
				return nil, err
			}
		case xml.Comment:
			continue
		default:
			return nil, errors.New("SVG contains unsupported XML content")
		}
	}
	if !seenRoot || depth != 0 {
		return nil, errors.New("SVG must contain one complete root element")
	}
	if err := encoder.Flush(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func unsafeSVGValue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(lower, "url(") {
		if !strings.HasPrefix(lower, "url(") || !strings.HasSuffix(lower, ")") {
			return true
		}
		return !strings.HasPrefix(strings.TrimSpace(lower[4:len(lower)-1]), "#")
	}
	return strings.Contains(lower, "data:") || strings.Contains(lower, "javascript:") || strings.Contains(lower, "http:") || strings.Contains(lower, "https:")
}
