package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
    bucketName = "images"
)

func main() {
    minioEndpoint := os.Getenv("MINIO_ENDPOINT")
    if minioEndpoint == "" {
        minioEndpoint = "localhost:9000"
    }
    log.Printf("Connecting to Minio at: %s", minioEndpoint)

    // Initialize Minio client
    minioClient, err := minio.New(minioEndpoint, &minio.Options{
        Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
        Secure: false,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create bucket if it doesnt exist
    err = createBucketIfNotExists(minioClient, bucketName)
    if err != nil {
        log.Fatal(err)
    }

    // Initialize Gin router
    r := gin.Default()
    r.Static("/static", "./static")
    
    // Routes
    r.GET("/presign", func(c *gin.Context) {
        filename := c.Query("filename")
        if filename == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
            return
        }

        // Generate presigned URL for upload
        presignedURL, err := minioClient.PresignedPutObject(context.Background(), bucketName, filename, time.Minute*15)
        if err != nil {
            log.Printf("Error generating presigned URL: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        log.Printf("Generated presigned URL for upload: %s", presignedURL.String())
        c.JSON(http.StatusOK, gin.H{
            "url": presignedURL.String(),
            "endpoint": minioEndpoint,
            "bucket": bucketName,
            "filename": filename,
        })
    })

    r.GET("/images", func(c *gin.Context) {
        objects := make([]string, 0)
        objectCh := minioClient.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{})
        
        for object := range objectCh {
            if object.Err != nil {
                log.Printf("Error listing object: %v", object.Err)
                continue
            }
            // Generate presigned URL for viewing
            presignedURL, err := minioClient.PresignedGetObject(context.Background(), bucketName, object.Key, time.Hour*24, nil)
            if err != nil {
                log.Printf("Error generating presigned URL for viewing: %v", err)
                continue
            }
            log.Printf("Generated presigned URL for viewing: %s", presignedURL.String())
            objects = append(objects, presignedURL.String())
        }

        c.JSON(http.StatusOK, gin.H{"images": objects})
    })

    fmt.Printf("Server starting on http://localhost:8080\n")
    r.Run(":8080")
}





func createBucketIfNotExists(minioClient *minio.Client, bucketName string) error {
    exists, err := minioClient.BucketExists(context.Background(), bucketName)
    if err != nil {
        return err
    }

    if !exists {
        log.Printf("Creating bucket: %s", bucketName)
        err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
        if err != nil {
            return err
        }
        log.Printf("Bucket created successfully")
    } else {
        log.Printf("Bucket already exists: %s", bucketName)
    }

    return nil
}