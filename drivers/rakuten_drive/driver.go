package rakuten_drive

import (
	"context"
	"fmt"
	"net/http"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/errs"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/go-resty/resty/v2"
)

type RakutenDrive struct {
	model.Storage
	Addition
}

func (d *RakutenDrive) Config() driver.Config {
	return config
}

func (d *RakutenDrive) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *RakutenDrive) Init(ctx context.Context) error {
	d.RootFolderPath = normalizeDirPath(d.RootFolderPath)
	return d.refreshToken()
}

func (d *RakutenDrive) Drop(ctx context.Context) error {
	return nil
}

func (d *RakutenDrive) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	files, err := d.getFiles(dir.GetPath())
	if err != nil {
		return nil, err
	}
	objs := make([]model.Obj, 0, len(files))
	for i := range files {
		objs = append(objs, &files[i])
	}
	return objs, nil
}

func (d *RakutenDrive) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	parent := parentDir(file.GetPath())
	filePath := baseName(file.GetPath())
	if f, ok := file.(*File); ok {
		parent = f.apiParentPath()
		filePath = f.apiPath()
	}
	var resp LinkResp
	_, err := d.request("/cloud/service/file/v1/filelink/download", http.MethodPost, func(req *resty.Request) {
		req.SetBody(base.Json{
			"host_id": d.UID,
			"path":    parent,
			"file": []base.Json{
				{
					"path": filePath,
					"size": file.GetSize(),
				},
			},
			"app_version": appVersion,
		})
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.URL == "" {
		return nil, fmt.Errorf("empty download url")
	}
	return &model.Link{URL: resp.URL}, nil
}

func (d *RakutenDrive) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	_, err := d.request("/cloud/service/file/v1/files/create", http.MethodPost, func(req *resty.Request) {
		req.SetContext(ctx).SetBody(base.Json{
			"host_id": d.UID,
			"name":    dirName,
			"path":    normalizeDirPath(parentDir.GetPath()),
		})
	}, nil)
	return err
}

func (d *RakutenDrive) Move(ctx context.Context, srcObj, dstDir model.Obj) error {
	return errs.NotImplement
}

func (d *RakutenDrive) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	return errs.NotImplement
}

func (d *RakutenDrive) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	return errs.NotImplement
}

func (d *RakutenDrive) Remove(ctx context.Context, obj model.Obj) error {
	parent := parentDir(obj.GetPath())
	filePath := baseName(obj.GetPath())
	versionID := obj.GetID()
	lastModified := formatDeleteTime(obj.ModTime())
	if f, ok := obj.(*File); ok {
		parent = f.apiParentPath()
		filePath = f.apiPath()
		versionID = f.VersionID
		lastModified = f.apiLastModified()
	}
	var resp DeleteResp
	_, err := d.request("/cloud/service/file/v3/files", http.MethodDelete, func(req *resty.Request) {
		req.SetBody(base.Json{
			"host_id": d.UID,
			"file": []base.Json{
				{
					"path":          filePath,
					"size":          obj.GetSize(),
					"version_id":    versionID,
					"last_modified": lastModified,
				},
			},
			"trash":  true,
			"prefix": parent,
		})
	}, &resp)
	if err != nil {
		return err
	}
	return d.waitTask(ctx, resp.Key)
}

func (d *RakutenDrive) Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) error {
	return d.upload(ctx, dstDir, file, up)
}

var _ driver.Driver = (*RakutenDrive)(nil)
