package rakuten_drive

import (
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
)

type RefreshTokenResp struct {
	UID          string `json:"uid"`
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
}

type ErrResp struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type File struct {
	ID              string    `json:"Id"`
	Path            string    `json:"Path"`
	Size            int64     `json:"Size"`
	IsFolder        bool      `json:"IsFolder"`
	Created         time.Time `json:"Created"`
	LastModified    time.Time `json:"LastModified"`
	Thumbnail       string    `json:"Thumbnail"`
	VersionID       string    `json:"VersionID"`
	OwnerID         string    `json:"OwnerID"`
	IsLatest        bool      `json:"IsLatest"`
	HasChildFolder  bool      `json:"HasChildFolder"`
	IsBackedUp      bool      `json:"IsBackedUp"`
	filePath        string
	parentPath      string
	lastModifiedRaw string
}

func (f *File) GetName() string {
	name := baseName(f.Path)
	if name == "" {
		name = baseName(f.filePath)
	}
	return strings.TrimSuffix(name, "/")
}

func (f *File) GetSize() int64 {
	return f.Size
}

func (f *File) ModTime() time.Time {
	return f.LastModified
}

func (f *File) CreateTime() time.Time {
	if f.Created.IsZero() {
		return f.ModTime()
	}
	return f.Created
}

func (f *File) IsDir() bool {
	return f.IsFolder
}

func (f *File) GetHash() utils.HashInfo {
	return utils.HashInfo{}
}

func (f *File) GetID() string {
	if f.ID != "" {
		return f.ID
	}
	return f.VersionID
}

func (f *File) GetPath() string {
	return f.filePath
}

func (f *File) Thumb() string {
	return f.Thumbnail
}

func (f *File) apiPath() string {
	if f.Path != "" {
		return f.Path
	}
	return baseName(f.filePath)
}

func (f *File) apiParentPath() string {
	if f.parentPath != "" {
		return f.parentPath
	}
	return parentDir(f.filePath)
}

func (f *File) apiLastModified() string {
	if f.lastModifiedRaw != "" {
		return f.lastModifiedRaw
	}
	if f.LastModified.IsZero() {
		return formatDeleteTime(time.Time{})
	}
	return formatDeleteTime(f.LastModified)
}

var _ model.Obj = (*File)(nil)
var _ model.Thumb = (*File)(nil)

type FilesResp struct {
	AccessLevel string `json:"access_level"`
	Count       int    `json:"count"`
	File        []File `json:"file"`
	LastPage    bool   `json:"last_page"`
	Owner       string `json:"owner"`
	Prefix      string `json:"prefix"`
	UsageSize   int64  `json:"usage_size"`
}

type LinkResp struct {
	URL string `json:"url"`
}

type DeleteResp struct {
	Key string `json:"key"`
}

type CheckResp struct {
	Action    string `json:"action"`
	State     string `json:"state"`
	UsageSize int64  `json:"usage_size"`
}

type UploadCheckResp struct {
	UploadID string       `json:"upload_id"`
	Prefix   string       `json:"prefix"`
	File     []UploadFile `json:"file"`
	Bucket   string       `json:"bucket"`
	Region   string       `json:"region"`
}

type UploadFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type UploadTokenResp struct {
	AccessKeyID     string    `json:"AccessKeyId"`
	Expiration      time.Time `json:"Expiration"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
}
