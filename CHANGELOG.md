## v1.1

### Added

- Cookies can be added to the HTTP request, either via the `Cookies` map or the `Cookie()` function
- Function `GetWithClient()` provides the ability to send the request with a custom HTTP client
- Function `FindStrict()` finds the first instance of the mentioned tag with the exact matching values of the provided attribute (previously `Find()`)
- Function `FindAllStrict()` finds all the instances of the mentioned tag with the exact matching values of the attributes (previously `FindAll()`)

## Changed

- Function `Find()` now finds the first instance of the mentioned tag with any matching values of the provided attribute.
- Function `FindAll()` now finds all the instances of the mentioned tag with any matching values of the provided attribute.

---

## v1.2

### Added

- Function `HasAttribute()` checks if the element has specific attribute
- Function `GetAttribute()` returns the value of specific attribute
- Function `FindParent()` returns the parent of the current element
- Function `Siblings()` returns array of Root struct
- Function `Children()` returns array of Root struct

## Changed

- Internal handling with DOM elements is replaced by Root structs (working only with structs if possible)
- Function `Attrs` renamed in `Attributes` 