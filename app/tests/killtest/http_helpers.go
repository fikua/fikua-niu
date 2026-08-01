package main

import (
	"bytes"
	"encoding/json"
	"io"
)

func jsonBody(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}

func decodeJSON(r io.Reader, out any) error {
	return json.NewDecoder(r).Decode(out)
}
