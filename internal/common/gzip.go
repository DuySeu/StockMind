package common

import (
	"compress/gzip"
	"io"
)

func GZIPCompression(body io.ReadCloser, header string) (io.ReadCloser, error) {
	switch header {
	case "gzip":
		gz, err := gzip.NewReader(body)
		if err != nil {
			return nil, err
		}
		return gz, nil
	default:
		return body, nil
	}
}
