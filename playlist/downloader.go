package playlist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/bgroupe/goicy/config"
	"github.com/bgroupe/goicy/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TODO: Base path input from config
// TODO: Implement progress bar: https://github.com/cheggaaa/pb
// TODO: Fix issue where session is created anyway

const (
	basePath  = "/tmp/goicy"
	writeMode = 0700
)

type FileDownloader struct {
	Session     string
	SessionPath string
	endpoint    string
	bucket      string
	private     bool `default:"false"`

	client *minio.Client
}

// Returns a new file downloader. Auth type is derived from members of a map.
// When downloading from private remote block storage, endpoint and bucket are saved in respective fields for compatibility with s3-flavored clients
func NewDownloader(cfg DownloadConfig) *FileDownloader {

	fd := FileDownloader{}

	// if err != nil {
	// 	panic(err)
	// }

	if cfg.Private {

		client, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds:  generateCredentialsFromConfig(),
			Secure: true,
		})

		if err != nil {
			panic(err)
		}

		fd = FileDownloader{
			client:   client,
			bucket:   cfg.Bucket,
			endpoint: cfg.Endpoint,
			private:  true,
		}
	}

	fd.createBasePathSession()

	return &fd
}

// Download file from public URL with no auth. Filename is derived by calling path.Base() on the full URL
func (fd *FileDownloader) DownloadPublicFile(fileUrl string) (string, error) {
	r, err := http.Get(fileUrl)

	if err != nil {
		return "whoops", err
	}
	if r.StatusCode != 200 {
		logger.Log("File not found on remote", 1)
	}
	defer r.Body.Close()

	filePath := path.Base(r.Request.URL.String())

	fullPath := fmt.Sprintf("%s/%s", fd.SessionPath, filePath)

	if _, err := os.Stat(fd.SessionPath); os.IsNotExist(err) {
		os.MkdirAll(fd.SessionPath, writeMode)
	}

	outputFile, err := os.Create(fullPath)

	if err != nil {
		return "file failed to load", err
	}

	defer outputFile.Close()

	_, err = io.Copy(outputFile, r.Body)

	spew.Dump(fullPath)

	return fullPath, err
}

// Download file from a private bucket. ACL configuration is assumed.
func (fd *FileDownloader) DownloadPrivateFile(objectPath string) (string, error) {
	reader, err := fd.client.GetObject(context.Background(), fd.bucket, objectPath, minio.GetObjectOptions{})

	if err != nil {
		logger.Log("Unable to get object", 1)
	}

	defer reader.Close()

	fullPath := fmt.Sprintf("%s/%s", fd.SessionPath, path.Base(objectPath))

	if _, err := os.Stat(fd.SessionPath); os.IsNotExist(err) {
		// DONE: return error here for write permission

		err := os.MkdirAll(fd.SessionPath, writeMode)
		if err != nil {
			panic(err)
		}
	}

	outputFile, err := os.Create(fullPath)

	if err != nil {
		return "file failed to load", err
	}

	defer outputFile.Close()

	stat, err := reader.Stat()
	if err != nil {
		return "failed to get object stat", err
	}

	if _, err := io.CopyN(outputFile, reader, stat.Size); err != nil {
		return "failed to copy object to file", err
	}

	return fullPath, err

}

// Delegates downloads to one of two functions depending on download config supplied to the constructor
func (fd *FileDownloader) Download(track Track) (string, error) {
	var (
		f   func(string) (string, error)
		dls string
	)

	if fd.private {
		f = fd.DownloadPrivateFile
		dls = track.ObjectPath
	} else {
		f = fd.DownloadPublicFile
		dls = track.Url
	}

	fp, err := f(dls)

	return fp, err
}

// Private
// Creates a session and appends to the base path to create a unique dir for download
func (fd *FileDownloader) createBasePathSession() {
	var bp string

	if config.Cfg.BasePath == "" {
		bp = basePath
	} else {
		bp = config.Cfg.BasePath
	}

	t := int32(time.Now().Unix())
	fd.Session = fmt.Sprintf("%v", t)
	fd.SessionPath = fmt.Sprintf("%s/%v", bp, t)
}

func generateCredentialsFromConfig() *credentials.Credentials {
	// DONE: Initialize auth type from config
	var (
		accessKey string
		secretKey string
	)

	if config.Cfg.AccessKey != "" && config.Cfg.SecretKey != "" {
		accessKey = config.Cfg.AccessKey
		secretKey = config.Cfg.SecretKey
	} else {
		switch config.Cfg.StorageType {
		case "DO", "do", "digitalocean":
			accessKey = os.Getenv("SPACES_ACCESS_TOKEN")
			secretKey = os.Getenv("SPACES_SECRET_KEY")
		case "S3", "s3":
			accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
			secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		default:
			accessKey = os.Getenv("SPACES_ACCESS_TOKEN")
			secretKey = os.Getenv("SPACES_SECRET_KEY")
		}
	}

	if accessKey == "" || secretKey == "" {
		panic("cannot find access credentials")
	}

	return credentials.NewStaticV4(accessKey, secretKey, "")
}

var ErrNoRoot = errors.New("MUST have administrator privileges")

// MustbeRoot returns an error message if the user is not root.
func MustbeRoot() error {
	if os.Getuid() != 0 {
		panic(ErrNoRoot)
	}
	return nil
}
