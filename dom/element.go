package dom

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
)

// Element represents a node in an XML document.
// Elements are arranged in a tree which corresponds to
// the structure of the XML documents.
type Element struct {
	Name     xml.Name
	children []*Element
	parent   *Element
	// Unlike a full-fledged XML DOM, we only have a single Content field
	// instead of representing Text nodes seperately.  We do not at present
	// support CDATA.
	Content    []byte
	Attributes []xml.Attr
}

// CreateElement creates a new element with the passed-in xml.Name.
func CreateElement(n xml.Name) *Element {
	element := &Element{Name: n}
	element.children = make([]*Element, 0, 5)
	element.Attributes = make([]xml.Attr, 0, 10)
	return element
}

// ElementN creates a new Element with a simple name.  The
// element will not be in a namespace until
// you put it in one by adding it to element.Name.Space.
func ElementN(n string) *Element {
	return CreateElement(xml.Name{Local: n})
}

// AddChild adds a new child element to this element.
// The child will be reparented if needed.
func (node *Element) AddChild(child *Element) {
	if child.parent != nil {
		child.parent.RemoveChild(child)
	}
	child.parent = node
	node.children = append(node.children, child)
}

// RemoveChild removes a child from this node.  The removed child
// will be returned.
func (node *Element) RemoveChild(child *Element) *Element {
	p := -1
	for i, v := range node.children {
		if v == child {
			p = i
			break
		}
	}

	if p == -1 {
		return node
	}

	copy(node.children[p:], node.children[p+1:])
	node.children = node.children[0 : len(node.children)-1]
	child.parent = nil
	return child
}

// Children returns all the children of the current node.
func (node *Element) Children() (res []*Element) {
	res = make([]*Element, 0, len(node.children))
	copy(res, node.children)
	return res
}

// Parent returns the parent of this node.
func (node *Element) Parent() *Element {
	return node.parent
}

// AddAttr adds a new attribute to this node.
// No checks are done to exclude duplicates to redefinition.
func (node *Element) AddAttr(attr xml.Attr) {
	node.Attributes = append(node.Attributes, attr)
}

// SetParent sets the new parent node for this node.
func (node *Element) SetParent(parent *Element) *Element {
	parent.AddChild(node)
	return node
}

func (node *Element) addNamespaces(encoder *Encoder) {
	// See if any of our attribs are in the xmlns namespace.
	// If they are, try to add them with their prefix
	for _, a := range node.Attributes {
		if a.Name.Space == "xmlns" {
			encoder.addNamespace(a.Value, a.Name.Local)
		}
	}

	encoder.addNamespace(node.Name.Space, "")
	for _, a := range node.Attributes {
		encoder.addNamespace(a.Name.Space, "")
	}
	for _, c := range node.children {
		c.addNamespaces(encoder)
	}
}

func namespacedName(e *Encoder, name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	if name.Space == "xmlns" {
		return name.Space + ":" + name.Local
	}
	prefix, found := e.nsURLMap[name.Space]
	if !found {
		log.Panicf("No prefix found in %v for namespace %s", e.nsURLMap, name.Space)
	}
	return prefix + ":" + name.Local
}

// Encode encodes an element using the passed-in Encoder.
func (node *Element) Encode(e *Encoder) (err error) {
	// This could use some refactoring. but it works Well Enough(tm)
	writeNamespaces := !e.started
	if writeNamespaces {
		node.addNamespaces(e)
		e.started = true
	}
	err = e.spaces()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(e, "<%s", namespacedName(e, node.Name))
	if err != nil {
		return err
	}
	for _, a := range node.Attributes {
		if a.Name.Space == "xmlns" {
			continue
		}
		_, err = fmt.Fprintf(e, " %s=\"%s\"", namespacedName(e, a.Name), a.Value)
		if err != nil {
			return err
		}
	}
	if writeNamespaces {
		for prefix, uri := range e.nsPrefixMap {
			_, err = fmt.Fprintf(e, " xmlns:%s=\"%s\"", prefix, uri)
			if err != nil {
				return err
			}
		}
	}
	if len(node.children) == 0 && len(node.Content) == 0 {
		ctag := "/>"
		if e.pretty {
			ctag = "/>\n"
		}
		_, err = e.WriteString(ctag)
		if err != nil {
			return err
		}
		return
	}
	_, err = e.WriteString(">")
	if len(node.Content) > 0 {
		xml.EscapeText(e, node.Content)
	}
	if len(node.children) > 0 {
		e.depth++
		if err = e.prettyEnd(); err != nil {
			return err
		}
		for _, c := range node.children {
			if err = c.Encode(e); err != nil {
				return err
			}
		}
		e.depth--
		if err = e.spaces(); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(e, "</%s>", namespacedName(e, node.Name))
	if err != nil {
		return err
	}
	return e.prettyEnd()
}

// Bytes returns a pretty-printed XML encoding of this part of the tree.
// The return is a byte array.
func (node *Element) Bytes() []byte {
	var b bytes.Buffer
	encoder := NewEncoder(&b)
	encoder.Pretty()
	node.Encode(encoder)
	encoder.Flush()
	return b.Bytes()
}

// String returns a pretty-printed XML encoding of this part of the tree.
//  The return is a string.
func (node *Element) String() string {
	return string(node.Bytes())
}
