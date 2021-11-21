package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const fileUrl = "https://oz-tf.nyc3.digitaloceanspaces.com/audio/private/raiments.mp3"
const endpoint = "nyc3.digitaloceanspaces.com"

var (
	bucket = "oz-tf"
	object = "audio/private/raiments.mp3"
)

func main() {
	accessKey := os.Getenv("SPACES_ACCESS_TOKEN")
	secretKey := os.Getenv("SPACES_SECRET_KEY")
	useSsl := true

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSsl,
	})

	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("%#v\n", client)

	reader, err := client.GetObject(context.Background(), bucket, object, minio.GetObjectOptions{})

	if err != nil {
		log.Fatalln(err)
	}

	defer reader.Close()

	localFile, err := os.Create("/tmp/raiments-copied.mp3")

	if err != nil {
		log.Fatalln(err)
	}

	defer localFile.Close()

	stat, err := reader.Stat()
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := io.CopyN(localFile, reader, stat.Size); err != nil {
		log.Fatalln(err)
	}

}
