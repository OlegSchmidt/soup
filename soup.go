/* soup package implements a simple web scraper for Go,
keeping it as similar as possible to BeautifulSoup
*/

package soup

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Root is a structure containing a pointer to an html node, the node value, and an error variable to return an error if occurred
type Root struct {
	Parent    *Root
	Pointer   *html.Node
	NodeValue string
	Error     error
}

var debug = false

// Headers contains all HTTP headers to send
var Headers = make(map[string]string)

// Cookies contains all HTTP cookies to send
var Cookies = make(map[string]string)

// SetDebug sets the debug status
// Setting this to true causes the panics to be thrown and logged onto the console.
// Setting this to false causes the errors to be saved in the Error field in the returned struct.
func SetDebug(d bool) {
	debug = d
}

// Header sets a new HTTP header
func Header(n string, v string) {
	Headers[n] = v
}

func Cookie(n string, v string) {
	Cookies[n] = v
}

// GetWithClient returns the HTML returned by the url using a provided HTTP client
func GetWithClient(url string, client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if debug {
			panic("Couldn't perform GET request to " + url)
		}
		return "", errors.New("couldn't perform GET request to " + url)
	}
	// Set headers
	for hName, hValue := range Headers {
		req.Header.Set(hName, hValue)
	}
	// Set cookies
	for cName, cValue := range Cookies {
		req.AddCookie(&http.Cookie{
			Name:  cName,
			Value: cValue,
		})
	}
	// Perform request
	resp, err := client.Do(req)
	if err != nil {
		if debug {
			panic("Couldn't perform GET request to " + url)
		}
		return "", errors.New("couldn't perform GET request to " + url)
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if debug {
			panic("Unable to read the response body")
		}
		return "", errors.New("unable to read the response body")
	}
	return string(bytes), nil
}

// Get returns the HTML returned by the url in string using the default HTTP client
func Get(url string) (string, error) {
	// Init a new HTTP client
	client := &http.Client{}
	return GetWithClient(url, client)
}

// HTMLParse parses the HTML returning a start pointer to the DOM
func HTMLParse(s string) Root {
	r, err := html.Parse(strings.NewReader(s))
	if err != nil {
		if debug {
			panic("Unable to parse the HTML")
		}
		return Root{nil, nil, "", errors.New("unable to parse the HTML")}
	}
	for r.Type != html.ElementNode {
		switch r.Type {
		case html.DocumentNode:
			r = r.FirstChild
		case html.DoctypeNode:
			r = r.NextSibling
		case html.CommentNode:
			r = r.NextSibling
		}
	}
	return Root{nil, r, r.Data, nil}
}

// Find finds the first occurrence of the given tag name,
// with or without attribute key and value specified,
// and returns a struct with a pointer to it
func (r Root) Find(args ...string) Root {
	result, ok := r.findOnce(args, false, false)
	if ok == false {
		if debug {
			panic("Element `" + args[0] + "` with attributes `" + strings.Join(args[1:], " ") + "` not found")
		}
		return Root{nil, nil, "", errors.New("element `" + args[0] + "` with attributes `" + strings.Join(args[1:], " ") + "` not found")}
	}
	return result
}

// FindStrict finds the first occurrence of the given tag name
// only if all the values of the provided attribute are an exact match
func (r Root) FindStrict(args ...string) Root {
	result, ok := r.findOnce(args, false, true)
	if ok == false {
		if debug {
			panic("Element `" + args[0] + "` with attributes `" + strings.Join(args[1:], " ") + "` not found")
		}
		return Root{nil, nil, "", errors.New("element `" + args[0] + "` with attributes `" + strings.Join(args[1:], " ") + "` not found")}
	}
	return result
}

// find the first matching element, loogs recursively into whole HTML tree beneath the given Root struct
func (r Root) findOnce(args []string, checkSelf bool, strict bool) (Root, bool) {
	var result Root
	success := false
	if checkSelf == true {
		matching := false
		switch len(args) {
		case 1:
			matching = elementMatching(r, strict, args[0], "", "")
			break
		case 3:
			matching = elementMatching(r, strict, args[0], args[1], args[2])
			break
		}
		if matching == true {
			result = r
			success = true
		}
	}
	if success == false {
		checkSelf = true
		children := r.Children()
		for position := range children {
			resultTemp, successTemp := children[position].findOnce(args, checkSelf, strict)
			if successTemp == true {
				result = resultTemp
				success = successTemp
				break
			}
		}
	}
	return result, success
}

// FindAllStrict finds all occurrences of the given tag name
// only if all the values of the provided attribute are an exact match
func (r Root) FindAll(args ...string) []Root {
	return r.findAll(args, false, false)
}

// FindAllStrict finds all occurrences of the given tag name
// only if all the values of the provided attribute are an exact match
func (r Root) FindAllStrict(args ...string) []Root {
	return r.findAll(args, false, true)
}

func (r Root) findAll(args []string, checkSelf bool, strict bool) []Root {
	var results []Root
	if checkSelf == true {
		matching := false
		switch len(args) {
		case 1:
			matching = elementMatching(r, strict, args[0], "", "")
			break
		case 3:
			matching = elementMatching(r, strict, args[0], args[1], args[2])
			break
		}
		if matching == true {
			results = append(results, r)
		}
	}

	children := r.Children()
	for position := range children {
		childResult := children[position].findAll(args, true, strict)
		for childResultPosition := range childResult {
			results = append(results, childResult[childResultPosition])
		}
	}

	return results
}

// FindNextSibling finds the next sibling of the pointer in the DOM
// returning a struct with a pointer to it
func (r Root) FindNextSibling() Root {
	nextSibling := r.Pointer.NextSibling
	if nextSibling == nil {
		if debug {
			panic("No next sibling found")
		}
		return Root{nil, nil, "", errors.New("no next sibling found")}
	}
	return Root{r.Parent, nextSibling, nextSibling.Data, nil}
}

// FindPrevSibling finds the previous sibling of the pointer in the DOM
// returning a struct with a pointer to it
func (r Root) FindPrevSibling() Root {
	prevSibling := r.Pointer.PrevSibling
	if prevSibling == nil {
		if debug {
			panic("No previous sibling found")
		}
		return Root{nil, nil, "", errors.New("no previous sibling found")}
	}
	return Root{r.Parent, prevSibling, prevSibling.Data, nil}
}

// FindNextElementSibling finds the next element sibling of the pointer in the DOM
// returning a struct with a pointer to it
func (r Root) FindNextElementSibling() Root {
	nextSibling := r.Pointer.NextSibling
	if nextSibling == nil {
		if debug {
			panic("No next element sibling found")
		}
		return Root{nil, nil, "", errors.New("no next element sibling found")}
	}
	if nextSibling.Type == html.ElementNode {
		return Root{r.Parent, nextSibling, nextSibling.Data, nil}
	}
	p := Root{r.Parent, nextSibling, nextSibling.Data, nil}
	return p.FindNextElementSibling()
}

// FindPrevElementSibling finds the previous element sibling of the pointer in the DOM
// returning a struct with a pointer to it
func (r Root) FindPrevElementSibling() Root {
	prevSibling := r.Pointer.PrevSibling
	if prevSibling == nil {
		if debug {
			panic("No previous element sibling found")
		}
		return Root{nil, nil, "", errors.New("no previous element sibling found")}
	}
	if prevSibling.Type == html.ElementNode {
		return Root{r.Parent, prevSibling, prevSibling.Data, nil}
	}
	p := Root{r.Parent, prevSibling, prevSibling.Data, nil}
	return p.FindPrevElementSibling()
}

// Children returns all direct children of this DOME element.
// passing true will make it possible to get all children, also the one's which are not html-nodes
func (r Root) Children(parameters ...bool) []Root {
	child := r.Pointer.FirstChild
	var children []Root
	for child != nil {
		if len(parameters) == 1 && parameters[0] == true || child.Type == html.ElementNode {
			children = append(children, Root{&r, child, child.Data, nil})
		}
		child = child.NextSibling
	}
	return children
}

// Siblings returns all siblings of this DOME element.
// passing true will make it possible to get all children, also the one's which are not html-nodes
func (r Root) Siblings(parameters ...bool) []Root {
	var siblings []Root

	for sibling := r.Pointer.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if len(parameters) == 1 && parameters[0] == true || sibling.Type == html.ElementNode {
			siblings = append(siblings, Root{r.Parent, sibling, sibling.Data, nil})
		}
	}

	return siblings
}

// FindParent returns the parent element
func (r Root) FindParent() Root {
	return Root{r.Parent.Parent, r.Parent.Pointer, r.Parent.NodeValue, nil}
}

// checks if the HTML Node has the given attribute
func (r Root) HasAttribute(attributeToFind string) bool {
	attributes := r.Attributes()
	found := false
	for attributeName := range attributes {
		if attributeName == attributeToFind {
			found = true
			break
		}
	}
	return found
}

// checks if the HTML Node has the given attribute
func (r Root) GetAttribute(attributeToFind string) string {
	attribute := ""
	if r.HasAttribute(attributeToFind) {
		attributes := r.Attributes()
		for attributeName := range attributes {
			if attributeName == attributeToFind {
				attribute = attributes[attributeName]
				break
			}
		}
	}

	return attribute
}

// Attrs returns a map containing all attributes
func (r Root) Attributes() map[string]string {
	if r.Pointer.Type != html.ElementNode {
		if debug {
			panic("Not an ElementNode")
		}
		return nil
	}
	if len(r.Pointer.Attr) == 0 {
		return nil
	}
	return getKeyValue(r.Pointer.Attr)
}

// Text returns the string inside a non-nested element
func (r Root) Text() string {
	k := r.Pointer.FirstChild
checkNode:
	if k != nil && k.Type != html.TextNode {
		k = k.NextSibling
		if k == nil {
			if debug {
				panic("No text node found")
			}
			return ""
		}
		goto checkNode
	}
	if k != nil {
		r, _ := regexp.Compile(`^\s+$`)
		if ok := r.MatchString(k.Data); ok {
			k = k.NextSibling
			if k == nil {
				if debug {
					panic("No text node found")
				}
				return ""
			}
			goto checkNode
		}
		return k.Data
	}
	return ""
}

// FullText returns the string inside even a nested element
func (r Root) FullText() string {
	var buf bytes.Buffer

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n == nil {
			return
		}
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		if n.Type == html.ElementNode {
			f(n.FirstChild)
		}
		if n.NextSibling != nil {
			f(n.NextSibling)
		}
	}

	f(r.Pointer.FirstChild)

	return buf.String()
}

// checks if the given root object is matching with the given filters
func elementMatching(r Root, strict bool, name string, nameAttribute string, valueAttribute string) bool {
	matching := false
	if r.Pointer.Type == html.ElementNode && ("" == name || r.NodeValue == name) {
		if nameAttribute == "" && valueAttribute == "" {
			matching = true
		} else if r.HasAttribute(nameAttribute) {
			matching = compareAttributeValues(strict, r.GetAttribute(nameAttribute), valueAttribute)
		}
	}
	return matching
}

// compares the string values with each other
// should it not be wanted as script, it will check if the valueToCheck-parts are all included in valueAttribute
func compareAttributeValues(strict bool, valueAttribute string, valueToCheck string) bool {
	matching := false
	if strict == true && valueAttribute == valueToCheck {
		matching = true
	} else if strict == false {
		attributeParts := strings.Fields(valueAttribute)
		searchParts := strings.Fields(valueToCheck)
		matching = true
		for positionSearch := range searchParts {
			found := false
			for positionAttribute := range attributeParts {
				if searchParts[positionSearch] == attributeParts[positionAttribute] {
					found = true
					break
				}
			}
			matching = matching == true && found == true
		}
	}
	return matching
}

// Returns a key pair value (like a dictionary) for each attribute
func getKeyValue(attributes []html.Attribute) map[string]string {
	var keyvalues = make(map[string]string)
	for i := 0; i < len(attributes); i++ {
		_, exists := keyvalues[attributes[i].Key]
		if exists == false {
			keyvalues[attributes[i].Key] = attributes[i].Val
		}
	}
	return keyvalues
}
