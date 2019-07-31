/* soup package implements a simple web scraper for Go,
keeping it as similar as possible to BeautifulSoup
*/

package soup

import (
	"bytes"
	"errors"
	"html"
	"io/ioutil"
	"log"
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
	result, ok := findOnce(r, args, false, false)
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
	result, ok := findOnce(r, args, false, true)
	if ok == false {
		if debug {
			panic("Element `" + args[0] + "` with attributes `" + strings.Join(args[1:], " ") + "` not found")
		}
		return Root{nil, nil, "", errors.New("element `" + args[0] + "` with attributes `" + strings.Join(args[1:], " ") + "` not found")}
	}
	return result
}

// FindAllStrict finds all occurrences of the given tag name
// only if all the values of the provided attribute are an exact match
func (r Root) FindAllStrict(args ...string) []Root {
	return r.findAll(args, false, true)
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

// Children retuns all direct children of this DOME element.
func (r Root) Children() []Root {
	child := r.Pointer.FirstChild
	var children []Root
	for child != nil {
		children = append(children, Root{&r, child, child.Data, nil})
		child = child.NextSibling
	}
	return children
}
func (r Root) Siblings() []Root {
	var siblings []Root

	for sibling := r.Pointer.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		siblings = append(siblings, Root{r.Parent, sibling, sibling.Data, nil})
	}

	return siblings
}

// Children retuns all direct children of this DOME element.
func (r Root) FindParent() Root {
	return Root{r.Parent, r.Parent.Pointer, r.Parent.Pointer.Data, nil}
}

// Attrs returns a map containing all attributes
func (r Root) Attrs() map[string]string {
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

func matchElementName(n *html.Node, name string) bool {
	return name == "" || name == n.Data
}

func elementMatching(r Root, name string, strict bool, nameAttribute string, valueAttribute string) bool {
	matching := false
	log.Output(2, "elementMatching params")
	log.Output(2, name)
	log.Output(2, nameAttribute)
	log.Output(2, valueAttribute)
	log.Output(2, "--------------------------")
	if r.Pointer.Type == html.ElementNode && matchElementName(r.Pointer, name) {
		if nameAttribute == "" && valueAttribute == "" {
			matching = true
		} else {
			for i := 0; i < len(r.Pointer.Attr); i++ {
				attribute := r.Pointer.Attr[i]
				if (strict && attributeAndValueEquals(attribute, nameAttribute, valueAttribute)) || (!strict && attributeContainsValue(attribute, nameAttribute, valueAttribute)) {
					matching = true
					break
				}
			}
		}
	}
	return matching
}

func findOnce(r Root, args []string, uni bool, strict bool) (Root, bool) {
	var result Root
	success := false
	for info := range args {
		log.Output(2, args[info])
	}
	if uni == true {
		matching := false
		switch len(args) {
		case 1:
			matching = elementMatching(r, args[0], true, "", "")
			break
		case 3:
			matching = elementMatching(r, args[0], true, args[1], args[2])
			break
		}
		if matching == true {
			result = r
			success = true
		}
	}
	if success == false {
		uni = true
		children := r.Children()
		for position := range children {
			resultTemp, successTemp := findOnce(children[position], args, uni, strict)
			if success == true {
				result = resultTemp
				success = successTemp
				break
			}
		}
	}
	return result, success
}

func (r Root) findAll(args []string, checkSelf bool, strict bool) []Root {
	var results []Root
	if checkSelf == true {
		matching := false
		switch len(args) {
		case 1:
			matching = elementMatching(r, args[0], true, "", "")
			break
		case 3:
			matching = elementMatching(r, args[0], true, args[1], args[2])
			break
		}
		if matching == true {
			results = append(results, r)
		}
	}

	siblings := r.Siblings()
	for position := range siblings {
		siblingResult := siblings[position].findAll(args, true, strict)
		for SiblingResultPosition := range siblingResult {
			results = append(results, siblingResult[SiblingResultPosition])
		}
	}

	return results
}

// attributeAndValueEquals reports when the html.Attribute attr has the same attribute name and value as from
// provided arguments
func attributeAndValueEquals(attr html.Attribute, attribute, value string) bool {
	return attr.Key == attribute && attr.Val == value
}

// attributeContainsValue reports when the html.Attribute attr has the same attribute name as from provided
// attribute argument and compares if it has the same value in its values parameter
func attributeContainsValue(attr html.Attribute, attribute, value string) bool {
	if attr.Key == attribute {
		for _, attrVal := range strings.Fields(attr.Val) {
			if attrVal == value {
				return true
			}
		}
	}
	return false
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
