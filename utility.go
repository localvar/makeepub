package main

import (
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func removeUtf8Bom(data []byte) []byte {
	if len(data) > 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	return data
}

func containsField(str, field string) bool {
	for _, f := range strings.Fields(str) {
		if f == field {
			return true
		}
	}
	return false
}

func isBlankNode(node *html.Node) bool {
	if node.Type == html.CommentNode {
		return true
	}
	if node.Type != html.TextNode {
		return false
	}
	isNonSpace := func(r rune) bool { return !unicode.IsSpace(r) }
	return strings.IndexFunc(node.Data, isNonSpace) == -1
}

func findFirstDirectChild(parent *html.Node, a atom.Atom) *html.Node {
	for node := parent.FirstChild; node != nil; node = node.NextSibling {
		if node.Type == html.ElementNode && node.DataAtom == a {
			return node
		}
	}
	return nil
}

func findDirectChildren(parent *html.Node, a atom.Atom) (result []*html.Node) {
	for node := parent.FirstChild; node != nil; node = node.NextSibling {
		if node.Type == html.ElementNode && node.DataAtom == a {
			result = append(result, node)
		}
	}
	return
}

func findFirstChild(parent *html.Node, a atom.Atom) *html.Node {
	for node := parent.FirstChild; node != nil; node = node.NextSibling {
		if node.Type != html.ElementNode {
			continue
		}
		if node.DataAtom == a {
			return node
		}
		if n := findFirstChild(node, a); n != nil {
			return n
		}
	}
	return nil
}

func findChildren(parent *html.Node, a atom.Atom) (result []*html.Node) {
	for node := parent.FirstChild; node != nil; node = node.NextSibling {
		if node.Type != html.ElementNode {
			continue
		}
		if node.DataAtom == a {
			result = append(result, node)
		}
		if r := findChildren(node, a); len(r) > 0 {
			result = append(result, r...)
		}
	}
	return
}

func findAttribute(node *html.Node, name string) *html.Attribute {
	for i := 0; i < len(node.Attr); i++ {
		if node.Attr[i].Key == name {
			return &node.Attr[i]
		}
	}
	return nil
}

func removeAttribute(node *html.Node, name string) {
	attr := node.Attr
	for i := len(attr) - 1; i >= 0; i-- {
		if attr[i].Key == name {
			attr = append(attr[:i], attr[i+1:]...)
		}
	}
	node.Attr = attr
}

func getAttributeValue(node *html.Node, name string, dflt string) string {
	if attr := findAttribute(node, name); attr != nil {
		return attr.Val
	}
	return dflt
}

func hasClass(node *html.Node, class string) bool {
	if attr := findAttribute(node, "class"); attr != nil {
		return containsField(attr.Val, class)
	}
	return false
}

func addClass(node *html.Node, class string) {
	if attr := findAttribute(node, "class"); attr != nil {
		if !containsField(attr.Val, class) {
			attr.Val = class + " " + attr.Val
		}
	} else {
		node.Attr = append(node.Attr, html.Attribute{Key: "class", Val: class})
	}
}

func removeClass(node *html.Node, class string) {
	attr := findAttribute(node, "class")
	if attr == nil {
		return
	}
	classes := ""
	for _, c := range strings.Fields(attr.Val) {
		if c == class {
			continue
		}
		if len(classes) == 0 {
			classes = c
		} else {
			classes = classes + " " + c
		}
	}
	attr.Val = classes
}

func cloneNode(node *html.Node) *html.Node {
	n := &html.Node{
		Type:     node.Type,
		DataAtom: node.DataAtom,
		Data:     node.Data,
		Attr:     make([]html.Attribute, len(node.Attr)),
	}
	copy(n.Attr, node.Attr)
	return n
}
