package playlist

import (
	"testing"
)

func TestDownloadType(t *testing.T) {
	dlc := DownloadConfig{
		Private:  false,
		Endpoint: "foo",
		Bucket:   "bar",
	}
	fd := NewDownloader(dlc)

	if fd.private {
		t.Fatalf("non privated downloader created")
	}
}

// TODO: need a test bucket
