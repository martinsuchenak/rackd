package storage

import (
	"encoding/json"
	"io"

	"github.com/pelletier/go-toml/v2"
)

// saveJSON saves data as JSON
func saveJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// loadJSON loads data from JSON
func loadJSON(r io.Reader, data interface{}) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(data)
}

// saveTOML saves data as TOML
func saveTOML(w io.Writer, data interface{}) error {
	encoder := toml.NewEncoder(w)
	return encoder.Encode(data)
}

// loadTOML loads data from TOML
func loadTOML(r io.Reader, data interface{}) error {
	decoder := toml.NewDecoder(r)
	return decoder.Decode(data)
}
