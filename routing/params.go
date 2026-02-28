package routing

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Params represents route parameters extracted from URL.
type Params map[string]string

// Get returns a parameter value by key.
func (p Params) Get(key string) string {
	return p[key]
}

// GetDefault returns a parameter value with a default.
func (p Params) GetDefault(key, defaultValue string) string {
	if val, ok := p[key]; ok {
		return val
	}
	return defaultValue
}

// Has checks if a parameter exists.
func (p Params) Has(key string) bool {
	_, ok := p[key]
	return ok
}

// Set sets a parameter value.
func (p Params) Set(key, value string) {
	p[key] = value
}

// Delete removes a parameter.
func (p Params) Delete(key string) {
	delete(p, key)
}

// Clone creates a copy of the params.
func (p Params) Clone() Params {
	newParams := make(Params, len(p))
	for k, v := range p {
		newParams[k] = v
	}
	return newParams
}

// Merge merges another Params into this one.
func (p Params) Merge(other Params) {
	for k, v := range other {
		p[k] = v
	}
}

// ToMap converts Params to a regular map.
func (p Params) ToMap() map[string]string {
	return map[string]string(p)
}

// ToJSON converts Params to JSON.
func (p Params) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// FromJSON creates Params from JSON.
func ParamsFromJSON(data []byte) (Params, error) {
	var p Params
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return p, nil
}

// Int returns a parameter as an int.
// Returns error if the parameter doesn't exist or cannot be parsed.
func (p Params) Int(key string) (int, error) {
	val, ok := p[key]
	if !ok {
		return 0, fmt.Errorf("parameter %s not found", key)
	}
	return strconv.Atoi(val)
}

// IntOk returns a parameter as an int with an existence check.
// Returns (value, true, nil) if found and valid.
// Returns (0, false, nil) if not found.
// Returns (0, true, error) if found but invalid.
func (p Params) IntOk(key string) (int, bool, error) {
	val, ok := p[key]
	if !ok {
		return 0, false, nil
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, true, err
	}
	return i, true, nil
}

// IntDefault returns a parameter as an int with a default.
// Returns defaultValue if the parameter doesn't exist or cannot be parsed.
func (p Params) IntDefault(key string, defaultValue int) int {
	val, ok := p[key]
	if !ok {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

// Int64 returns a parameter as an int64.
func (p Params) Int64(key string) (int64, error) {
	val, ok := p[key]
	if !ok {
		return 0, fmt.Errorf("parameter %s not found", key)
	}
	return strconv.ParseInt(val, 10, 64)
}

// Int64Ok returns a parameter as an int64 with an existence check.
func (p Params) Int64Ok(key string) (int64, bool, error) {
	val, ok := p[key]
	if !ok {
		return 0, false, nil
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, true, err
	}
	return i, true, nil
}

// Float64 returns a parameter as a float64.
func (p Params) Float64(key string) (float64, error) {
	val, ok := p[key]
	if !ok {
		return 0, fmt.Errorf("parameter %s not found", key)
	}
	return strconv.ParseFloat(val, 64)
}

// Float64Ok returns a parameter as a float64 with an existence check.
func (p Params) Float64Ok(key string) (float64, bool, error) {
	val, ok := p[key]
	if !ok {
		return 0, false, nil
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, true, err
	}
	return f, true, nil
}

// Bool returns a parameter as a bool.
func (p Params) Bool(key string) (bool, error) {
	val, ok := p[key]
	if !ok {
		return false, fmt.Errorf("parameter %s not found", key)
	}
	return strconv.ParseBool(val)
}

// BoolOk returns a parameter as a bool with an existence check.
func (p Params) BoolOk(key string) (bool, bool, error) {
	val, ok := p[key]
	if !ok {
		return false, false, nil
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false, true, err
	}
	return b, true, nil
}

// Slice returns a parameter as a slice (for catch-all params).
func (p Params) Slice(key string) []string {
	val := p[key]
	if val == "" {
		return nil
	}
	return strings.Split(val, "/")
}

// ParamExtractor extracts parameters from URLs.
type ParamExtractor struct {
	pattern string
	regex   *regexp.Regexp
	params  []string
}

// NewParamExtractor creates a new parameter extractor for a pattern.
func NewParamExtractor(pattern string) *ParamExtractor {
	pe := &ParamExtractor{
		pattern: pattern,
		params:  make([]string, 0),
	}

	// Build regex from pattern
	pe.buildRegex()

	return pe
}

// buildRegex builds a regex from the route pattern.
func (pe *ParamExtractor) buildRegex() {
	segments := strings.Split(strings.Trim(pe.pattern, "/"), "/")
	regexParts := make([]string, 0, len(segments))

	for _, seg := range segments {
		if seg == "" {
			continue
		}

		// Catch-all parameter
		if strings.HasPrefix(seg, "*") {
			paramName := seg[1:]
			pe.params = append(pe.params, paramName)
			regexParts = append(regexParts, "(.*)")
			continue
		}

		// Dynamic parameter
		if strings.HasPrefix(seg, ":") {
			paramName := seg[1:]
			pattern := "([^/]+)"

			// Custom regex validation support (e.g., :id<[0-9]+>)
			if startIdx := strings.Index(paramName, "<"); startIdx != -1 && strings.HasSuffix(paramName, ">") {
				customPattern := paramName[startIdx+1 : len(paramName)-1]
				// Validate the custom regex pattern
				if _, err := regexp.Compile(customPattern); err == nil {
					pattern = "(" + customPattern + ")"
				}
				// If invalid, fall back to default pattern "([^/]+)"
				paramName = paramName[:startIdx]
			}

			pe.params = append(pe.params, paramName)
			regexParts = append(regexParts, pattern)
			continue
		}

		// Static segment - escape special regex characters
		regexParts = append(regexParts, regexp.QuoteMeta(seg))
	}

	// Build final regex
	pattern := "^/" + strings.Join(regexParts, "/") + "$"
	pe.regex = regexp.MustCompile(pattern)
}

// Extract extracts parameters from a URL path.
func (pe *ParamExtractor) Extract(path string) (Params, bool) {
	matches := pe.regex.FindStringSubmatch(path)
	if matches == nil {
		return nil, false
	}

	params := make(Params)
	for i, paramName := range pe.params {
		if i+1 < len(matches) {
			params[paramName] = matches[i+1]
		}
	}

	return params, true
}

// Match checks if a path matches the pattern.
func (pe *ParamExtractor) Match(path string) bool {
	return pe.regex.MatchString(path)
}

// Params returns the parameter names for this pattern.
func (pe *ParamExtractor) Params() []string {
	return pe.params
}

// QueryParams represents URL query parameters.
type QueryParams struct {
	values url.Values
}

// NewQueryParams creates QueryParams from a URL query string.
func NewQueryParams(query string) *QueryParams {
	values, _ := url.ParseQuery(query)
	return &QueryParams{values: values}
}

// NewQueryParamsFromValues creates QueryParams from url.Values.
func NewQueryParamsFromValues(values url.Values) *QueryParams {
	return &QueryParams{values: values}
}

// Get returns the first value for a key.
func (qp *QueryParams) Get(key string) string {
	return qp.values.Get(key)
}

// GetAll returns all values for a key.
func (qp *QueryParams) GetAll(key string) []string {
	return qp.values[key]
}

// Has checks if a key exists.
func (qp *QueryParams) Has(key string) bool {
	return qp.values.Has(key)
}

// Set sets a value for a key.
func (qp *QueryParams) Set(key, value string) {
	qp.values.Set(key, value)
}

// Add adds a value for a key.
func (qp *QueryParams) Add(key, value string) {
	qp.values.Add(key, value)
}

// Del deletes a key.
func (qp *QueryParams) Del(key string) {
	qp.values.Del(key)
}

// Int returns a query parameter as an int.
func (qp *QueryParams) Int(key string) (int, error) {
	val := qp.values.Get(key)
	if val == "" {
		return 0, fmt.Errorf("query parameter %s not found", key)
	}
	return strconv.Atoi(val)
}

// IntOk returns a query parameter as an int with an existence check.
func (qp *QueryParams) IntOk(key string) (int, bool, error) {
	val := qp.values.Get(key)
	if val == "" {
		return 0, false, nil
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, true, err
	}
	return i, true, nil
}

// IntDefault returns a query parameter as an int with a default.
func (qp *QueryParams) IntDefault(key string, defaultValue int) int {
	val := qp.values.Get(key)
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

// Bool returns a query parameter as a bool.
func (qp *QueryParams) Bool(key string) (bool, error) {
	val := qp.values.Get(key)
	if val == "" {
		return false, fmt.Errorf("query parameter %s not found", key)
	}
	return strconv.ParseBool(val)
}

// BoolOk returns a query parameter as a bool with an existence check.
func (qp *QueryParams) BoolOk(key string) (bool, bool, error) {
	val := qp.values.Get(key)
	if val == "" {
		return false, false, nil
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false, true, err
	}
	return b, true, nil
}

// BoolDefault returns a query parameter as a bool with a default.
func (qp *QueryParams) BoolDefault(key string, defaultValue bool) bool {
	val := qp.values.Get(key)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return b
}

// ToMap converts to a regular map.
func (qp *QueryParams) ToMap() map[string][]string {
	return qp.values
}

// Encode encodes the query parameters to a string.
func (qp *QueryParams) Encode() string {
	return qp.values.Encode()
}

// RouteMatch represents a matched route with parameters.
type RouteMatch struct {
	// Route is the matched route
	Route *Route
	// PathParams are the path parameters
	PathParams Params
	// QueryParams are the query parameters
	QueryParams *QueryParams
	// URL is the full URL
	URL *url.URL
}

// NewRouteMatch creates a new route match.
func NewRouteMatch(route *Route, path string) (*RouteMatch, error) {
	// Parse URL
	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Extract path params
	extractor := NewParamExtractor(route.Path)
	pathParams, ok := extractor.Extract(u.Path)
	if !ok {
		return nil, fmt.Errorf("path does not match route pattern")
	}

	return &RouteMatch{
		Route:       route,
		PathParams:  pathParams,
		QueryParams: NewQueryParams(u.RawQuery),
		URL:         u,
	}, nil
}

// Param returns a path parameter value.
func (rm *RouteMatch) Param(key string) string {
	return rm.PathParams.Get(key)
}

// Query returns a query parameter value.
func (rm *RouteMatch) Query(key string) string {
	return rm.QueryParams.Get(key)
}

// BuildURL builds a URL from a route pattern and parameters.
func BuildURL(pattern string, pathParams Params, queryParams *QueryParams) string {
	// Replace path parameters
	url := pattern
	for key, value := range pathParams {
		url = strings.ReplaceAll(url, ":"+key, value)
		url = strings.ReplaceAll(url, "*"+key, value)
	}

	// Add query parameters
	if queryParams != nil && len(queryParams.values) > 0 {
		url += "?" + queryParams.Encode()
	}

	return url
}

// PathBuilder helps build paths with parameters.
type PathBuilder struct {
	pattern string
	params  Params
	query   url.Values
}

// NewPathBuilder creates a new path builder.
func NewPathBuilder(pattern string) *PathBuilder {
	return &PathBuilder{
		pattern: pattern,
		params:  make(Params),
		query:   make(url.Values),
	}
}

// Param sets a path parameter.
func (pb *PathBuilder) Param(key, value string) *PathBuilder {
	pb.params[key] = value
	return pb
}

// Query sets a query parameter.
func (pb *PathBuilder) Query(key, value string) *PathBuilder {
	pb.query.Set(key, value)
	return pb
}

// QueryAdd adds a query parameter.
func (pb *PathBuilder) QueryAdd(key, value string) *PathBuilder {
	pb.query.Add(key, value)
	return pb
}

// Build builds the final URL.
func (pb *PathBuilder) Build() string {
	url := pb.pattern

	// Replace path parameters
	for key, value := range pb.params {
		url = strings.ReplaceAll(url, ":"+key, value)
		url = strings.ReplaceAll(url, "*"+key, value)
	}

	// Add query parameters
	if len(pb.query) > 0 {
		url += "?" + pb.query.Encode()
	}

	return url
}

// ValidateParams validates route parameters against a route.
func ValidateParams(route *Route, params Params) error {
	for _, param := range route.Params {
		if _, ok := params[param]; !ok {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}
	return nil
}
