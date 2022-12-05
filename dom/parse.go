package dom

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

const TooManyRootElements = "More than one root Element not allowed!"

func parseElement(decoder *xml.Decoder, tok xml.StartElement) (res *Element, err error) {
	res = CreateElement(tok.Name)
	for _, attr := range tok.Attr {
		res.AddAttr(attr)
	}

	for {
		newtok, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch rt := newtok.(type) {
		case xml.EndElement:
			return res, nil
		case xml.CharData:
			content := bytes.TrimSpace([]byte(rt.Copy()))
			if len(content) > 0 {
				res.Content = content
			}
		case xml.StartElement:
			child, err := parseElement(decoder, rt)
			if err != nil {
				return nil, err
			}
			res.AddChild(child)
		}
	}
}

// ParseOptions specifies some parsing options.
type ParseOptions struct {
	CharsetReader func(string, io.Reader)(io.Reader,error)
}

func defaultOptions() *ParseOptions {
	return &ParseOptions{
		CharsetReader: func(s string, r io.Reader)(io.Reader,error){ return r,nil },
	}
}

// ParseElements parses the XML elements in the passed io.Reader
// and returns an array of parsed Elements and an error.  If error
// is not nil, then all the elements in the Reader were parsed
// corrently.
//
// This assumes our input is always UTF-8, no matter what lies
// the <?xml?> header says.
func ParseElements(r io.Reader) (elements []*Element, err error) {
	return ParseElementsWithOptions(r, defaultOptions())
}

// ParseElementsWithCharsetReader is like ParseElements but more options can
// be specified.
func ParseElementsWithOptions(r io.Reader, opts *ParseOptions) (elements []*Element, err error) {
	if opts == nil {
		opts = defaultOptions()
	}
	decoder := xml.NewDecoder(r)
	decoder.Strict = true
	decoder.CharsetReader = opts.CharsetReader
	elements = []*Element{}
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return elements, err
		}
		switch rt := tok.(type) {
		case xml.StartElement:
			element, err := parseElement(decoder, rt)
			if err != nil {
				return elements, err
			}
			elements = append(elements, element)
		}
	}
	return elements, nil
}

// Parse parses the XML document from the passed io.Reader and
// returns either a Document or an error if the io.Reader stream
// could not be parsed as a well-formed XML document.
func Parse(r io.Reader) (doc *Document, err error) {
	return ParseWithOptions(r, defaultOptions())
}

// ParseWithOptions is like Parse but more options can be specified.
func ParseWithOptions(r io.Reader, opts *ParseOptions) (doc *Document, err error) {
	elements, err := ParseElementsWithOptions(r, opts)
	if err != nil {
		return nil, err
	}
	if len(elements) > 1 {
		return nil, errors.New(TooManyRootElements)
	}
	doc = CreateDocument()
	if len(elements) == 1 {
		doc.SetRoot(elements[0])
	}
	return doc, nil
}
