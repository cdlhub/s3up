package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/aws/aws-sdk-go/aws/session"
)

var files []string

// command line flags
var (
	dir       string
	createDir bool
	region    string
	bucket    string
	profile   string
)

func printPath(path string, f os.FileInfo, err error) error {
	if !f.IsDir() {
		println(path)
	}
	return nil
}

func addFile(path string, f os.FileInfo, err error) error {
	if err != nil {
		return fmt.Errorf("unable to parse %q: %v", path, err)
	}
	if f.IsDir() {
		return nil
	}

	files = append(files, path)
	return nil
}

func upload(uploader *s3manager.Uploader, filename string, bucket string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("unable to open file %q: %v", filename, err)
	}
	defer file.Close()

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("unable to upload %q to %q: %v", filename, bucket, err)
	}

	return nil
}

func init() {
	flag.StringVar(&region, "region", "eu-west-3", "bucket region")
	flag.StringVar(&bucket, "bucket", "", "name of the bucket to uplaod files to")
	flag.StringVar(&profile, "profile", "", "AWS profile name for credentials")
	flag.BoolVar(&createDir, "createDir", false, "set to true to create base directory in the bucket")
	flag.Parse()

	dir = flag.Arg(0)
}

// setwd sets program working directory.
func setwd(path string, createDir bool) error {
	if createDir {
		path = filepath.Join(path, "..")
	}

	if err := os.Chdir(path); err != nil {
		return err
	}

	return nil
}

func main() {
	// TODO: deal logically with "." argument
	// TODO: add -exclude
	// TODO: gitify
	// TODO: add retry
	// TODO: add stop/resume

	if err := setwd(dir, createDir); err != nil {
		log.Fatalf("unable to change working directory to %q: %v", dir, err)
	}

	dirName := "."
	if createDir {
		dirName = filepath.Base(dir)
	}

	if err := filepath.Walk(dirName, addFile); err != nil {
		err = fmt.Errorf("cannot parse directory %q: %v", dir, err)
		log.Fatalln(err)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config:  aws.Config{Region: aws.String(region)},
		Profile: profile,
	}))
	uploader := s3manager.NewUploader(sess)

	nerr := 0
	for _, f := range files {
		log.Printf("Uploading %q\n", f)
		err := upload(uploader, f, bucket)
		if err != nil {
			nerr++
			log.Println(err)
		}
	}

	if nerr > 0 {
		log.Println("[FAIL] not all files could be uploaded")
	} else {
		log.Println("[OK] uploads succeeded")
	}
}
