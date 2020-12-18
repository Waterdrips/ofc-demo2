package function

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go"
	"github.com/openfaas/openfaas-cloud/sdk"
)

const imgPath = "image/image.png"

func Handle(w http.ResponseWriter, r *http.Request) {
	region := regionName()

	bucketName := bucketName()

	minioClient, connectErr := connectToMinio()
	if connectErr != nil {
		log.Printf("S3/Minio connection error %s\n", connectErr.Error())
		w.Write([]byte(connectErr.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodPost:

		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		r.ParseMultipartForm(10 << 20)
		// FormFile returns the first file for the given key `myFile`
		// it also returns the FileHeader so we can get the Filename,
		// the Header and the size of the file
		file, handler, err := r.FormFile("image")
		if err != nil {
			fmt.Println("Error Retrieving the File")
			fmt.Println(err)
			return
		}
		defer file.Close()
		fmt.Printf("Uploaded File: %+v\n", handler.Filename)
		fmt.Printf("File Size: %+v\n", handler.Size)
		fmt.Printf("MIME Header: %+v\n", handler.Header)

		minioClient.MakeBucket(bucketName, region)

		defer r.Body.Close()
		_, err = minioClient.PutObject(bucketName,
			imgPath,
			file,
			r.ContentLength,
			minio.PutObjectOptions{})

		if err != nil {
			log.Printf("error writing: %s, error: %s", imgPath, err.Error())
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return

	case http.MethodGet:
		obj, err := minioClient.GetObject(bucketName, imgPath, minio.GetObjectOptions{})

		if err != nil {
			log.Printf("error getting: %s, error: %s", imgPath, err.Error())
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		imageBytes, err := ioutil.ReadAll(obj)
		if err != nil {
			log.Printf("error reading: %s, error: %s", imgPath, err.Error())
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("content-type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(imageBytes)
		return
	}

	return
}

func bucketName() string {
	bucketName, exist := os.LookupEnv("s3_bucket")
	if exist == false || len(bucketName) == 0 {
		bucketName = "pipeline"
		log.Printf("Bucket name not found, set to default: %v\n", bucketName)
	}
	return bucketName
}

func regionName() string {
	regionName, exist := os.LookupEnv("s3_region")
	if exist == false || len(regionName) == 0 {
		regionName = "us-east-1"
	}
	return regionName
}

func connectToMinio() (*minio.Client, error) {

	endpoint := os.Getenv("s3_url")

	tlsEnabled := tlsEnabled()

	secretKey, _ := sdk.ReadSecret("s3-secret-key")
	accessKey, _ := sdk.ReadSecret("s3-access-key")

	return minio.New(endpoint, accessKey, secretKey, tlsEnabled)
}

func tlsEnabled() bool {
	if connection := os.Getenv("s3_tls"); connection == "true" || connection == "1" {
		return true
	}
	return false
}
