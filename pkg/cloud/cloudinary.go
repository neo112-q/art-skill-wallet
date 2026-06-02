package cloud

import (
	"context"
	"mime/multipart"
	"os"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// UploadResult holds the URL and public ID returned by Cloudinary.
type UploadResult struct {
	URL      string
	PublicID string
}

// newClient creates a Cloudinary client from environment variables.
func newClient() (*cloudinary.Cloudinary, error) {
	return cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
}

// UploadFile uploads a multipart file to Cloudinary and returns the URL and public ID.
// folder: the Cloudinary folder to upload into (e.g. "artworks", "avatars", "proofs")
func UploadFile(file multipart.File, folder string) (UploadResult, error) {
	cld, err := newClient()
	if err != nil {
		return UploadResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:         folder,
		ResourceType:   "auto", // handles images and videos
	})
	if err != nil {
		return UploadResult{}, err
	}

	return UploadResult{
		URL:      res.SecureURL,
		PublicID: res.PublicID,
	}, nil
}

// DeleteFile removes a file from Cloudinary by its public ID.
func DeleteFile(publicID string) error {
	if publicID == "" {
		return nil
	}

	cld, err := newClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	return err
}
