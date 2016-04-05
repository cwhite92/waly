package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys the provided directory",
	Long: `Uploads all files from the provided directory. This command uploads
	deeply nested files.`,
	Run: func(cmd *cobra.Command, args []string) {
		deploy(cmd, args)
	},
}

var source string
var bucket string
var key string
var secret string
var region string

func init() {
	RootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&source, "source", "s", "", "Source directory to read from")
	deployCmd.Flags().StringVarP(&bucket, "bucket", "b", "", "S3 bucket to deploy to")
	deployCmd.Flags().StringVarP(&key, "key", "k", "", "AWS key")
	deployCmd.Flags().StringVarP(&secret, "secret", "x", "", "AWS secret")
	deployCmd.Flags().StringVarP(&region, "region", "r", "eu-west-1", "AWS region")
}

func deploy(cmd *cobra.Command, args []string) {
	if source == "" {
		log.Fatal("Source directory must be provided")
	}

	if bucket == "" {
		log.Fatal("Bucket name must be provided")
	}

	exists, _ := dirExists(source)

	if !exists {
		log.Fatal("Source directory does not exist")
	}

	svc := s3.New(session.New(&aws.Config{Region: aws.String(region), Credentials: credentials.NewStaticCredentials(key, secret, "")}))

	exists, _ = bucketExists(svc, bucket)

	if !exists {
		log.Fatal("Destination bucket does not exist")
	}

	prefix := buildPrefix()

	filepath.Walk(source, upload(svc, bucket, prefix))
	fmt.Printf("Uploaded site with prefix: %s\n", prefix)
}

func upload(svc *s3.S3, bucket string, prefix string) filepath.WalkFunc {
	return func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}

		file, err := os.Open(path)

		if err != nil {
			log.Printf("Unable to open %s", path)
		}

		defer file.Close()

		key := fmt.Sprintf("%s/%s", prefix, strings.Replace(path, source, "", -1))

		_, err = svc.PutObject(&s3.PutObjectInput{
			Body:   file,
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})

		if err != nil {
			log.Printf("Failed to upload %s", path)
		}

		fmt.Printf("Uploaded %s\n", path)

		return nil
	}
}

func buildPrefix() string {
	t := time.Now()

	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d", t.Year(), t.Month(),
		t.Day(), t.Hour(), t.Minute(), t.Second())
}

func bucketExists(svc *s3.S3, name string) (bool, error) {
	result, err := svc.ListBuckets(&s3.ListBucketsInput{})

	if err != nil {
		return false, err
	}

	for _, bucket := range result.Buckets {
		if aws.StringValue(bucket.Name) == name {
			return true, nil
		}
	}

	return false, nil
}

func dirExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}
