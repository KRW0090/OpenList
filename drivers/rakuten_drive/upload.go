package rakuten_drive

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-resty/resty/v2"
)

const uploadPartSize = 5 * 1024 * 1024

func (d *RakutenDrive) upload(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) error {
	parent := normalizeDirPath(dstDir.GetPath())
	check, err := d.checkUpload(ctx, parent, file)
	if err != nil {
		return err
	}
	token, err := d.getUploadToken(ctx)
	if err != nil {
		return err
	}
	if err = validateUploadData(check, token); err != nil {
		return err
	}

	objectKey := strings.TrimPrefix(check.Prefix+check.File[0].Path, "/")
	client, err := newUploadClient(check.Region, token)
	if err != nil {
		return err
	}
	if err = uploadS3Object(ctx, client, check.Bucket, objectKey, file, up); err != nil {
		return err
	}
	if err = d.waitTask(ctx, check.UploadID); err != nil {
		return err
	}
	return d.completeUpload(ctx, parent, check.File[0])
}

func (d *RakutenDrive) checkUpload(ctx context.Context, parent string, file model.FileStreamer) (*UploadCheckResp, error) {
	var resp UploadCheckResp
	_, err := d.request("/cloud/service/file/v1/check/upload", http.MethodPost, func(req *resty.Request) {
		req.SetContext(ctx).SetBody(base.Json{
			"host_id": d.UID,
			"file": []base.Json{{
				"path": file.GetName(),
				"size": file.GetSize(),
			}},
			"path":      parent,
			"upload_id": "",
			"replace":   false,
		})
	}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (d *RakutenDrive) getUploadToken(ctx context.Context) (*UploadTokenResp, error) {
	var resp UploadTokenResp
	_, err := d.request("/cloud/service/file/v1/filelink/token", http.MethodGet, func(req *resty.Request) {
		req.SetContext(ctx).SetQueryParams(map[string]string{
			"host_id": d.UID,
			"path":    "hello",
		})
	}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func validateUploadData(check *UploadCheckResp, token *UploadTokenResp) error {
	if check.UploadID == "" || check.Bucket == "" || check.Region == "" || len(check.File) != 1 || check.File[0].Path == "" {
		return fmt.Errorf("invalid upload check response")
	}
	if token.AccessKeyID == "" || token.SecretAccessKey == "" || token.SessionToken == "" {
		return fmt.Errorf("invalid upload token response")
	}
	return nil
}

func newUploadClient(region string, token *UploadTokenResp) (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(token.AccessKeyID, token.SecretAccessKey, token.SessionToken),
		Region:      aws.String(region),
		HTTPClient:  base.HttpClient,
	})
	if err != nil {
		return nil, err
	}
	return s3.New(sess), nil
}

func uploadS3Object(ctx context.Context, client *s3.S3, bucket, key string, file model.FileStreamer, up driver.UpdateProgress) error {
	contentType := file.GetMimetype()
	uploader := s3manager.NewUploaderWithClient(client, func(u *s3manager.Uploader) {
		u.PartSize = uploadPartSize
		u.Concurrency = 4
	})
	reader := driver.NewLimitedUploadStream(ctx, &driver.ReaderUpdatingProgress{
		Reader:         file,
		UpdateProgress: up,
	})
	_, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	return err
}

func (d *RakutenDrive) completeUpload(ctx context.Context, parent string, file UploadFile) error {
	_, err := d.request("/cloud/service/file/v1/complete/upload", http.MethodPost, func(req *resty.Request) {
		req.SetContext(ctx).SetBody(base.Json{
			"host_id": d.UID,
			"file": []base.Json{{
				"path": file.Path,
				"size": file.Size,
			}},
			"path":  parent,
			"state": "complete",
		})
	}, nil)
	return err
}
