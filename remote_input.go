package gospa

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrJSONTooDeep is returned when remote action JSON exceeds [remoteJSONMaxNesting].
var ErrJSONTooDeep = errors.New("json nesting exceeds maximum")

const remoteJSONMaxNesting = 64

// decodeRemoteActionBody parses JSON for remote actions with bounded nesting depth and
// json.Number for numeric values (avoids float64 surprises for large integers).
func decodeRemoteActionBody(body []byte) (interface{}, error) {
	if err := validateJSONMaxNesting(body, remoteJSONMaxNesting); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func validateJSONMaxNesting(data []byte, maxDepth int) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	var depth int
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			if depth != 0 {
				return fmt.Errorf("invalid json: unbalanced delimiters")
			}
			return nil
		}
		if err != nil {
			return err
		}
		if t, ok := tok.(json.Delim); ok {
			switch t {
			case '{', '[':
				depth++
				if depth > maxDepth {
					return fmt.Errorf("%w: max %d", ErrJSONTooDeep, maxDepth)
				}
			case '}', ']':
				if depth > 0 {
					depth--
				}
			}
		}
	}
}
