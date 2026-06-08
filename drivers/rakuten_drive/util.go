package rakuten_drive

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/go-resty/resty/v2"
)

const (
	apiBase    = "https://api.rakuten-drive.com"
	appVersion = "v21.11.10"
)

func normalizePath(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	p = strings.TrimPrefix(p, "/")
	if p == "." {
		return ""
	}
	return p
}

func normalizeDirPath(p string) string {
	p = normalizePath(p)
	if p != "" && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

func parentDir(p string) string {
	p = normalizePath(p)
	p = strings.TrimSuffix(p, "/")
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return ""
	}
	return p[:i+1]
}

func baseName(p string) string {
	p = normalizePath(p)
	if strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
		return p[strings.LastIndex(p, "/")+1:] + "/"
	}
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return p
	}
	return p[i+1:]
}

func joinAPIPath(parent, name string, isFolder bool) string {
	parent = normalizeDirPath(parent)
	name = strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/")
	full := parent + name
	if isFolder && !strings.HasSuffix(full, "/") {
		full += "/"
	}
	return full
}

func resolveFilePath(parent, filePath string, isFolder bool) string {
	parent = normalizeDirPath(parent)
	filePath = normalizePath(filePath)
	if parent == "" || strings.HasPrefix(filePath, parent) {
		if isFolder && filePath != "" && !strings.HasSuffix(filePath, "/") {
			filePath += "/"
		}
		return filePath
	}
	return joinAPIPath(parent, filePath, isFolder)
}

func formatDeleteTime(t time.Time) string {
	if t.IsZero() {
		return time.UnixMilli(0).UTC().Format("2006-01-02T15:04:05.000Z")
	}
	return time.UnixMilli(t.Unix()).UTC().Format("2006-01-02T15:04:05.000Z")
}

func (d *RakutenDrive) refreshToken() error {
	var resp RefreshTokenResp
	res, err := base.RestyClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(base.Json{
			"refresh_token": d.RefreshToken,
		}).
		SetResult(&resp).
		Post(apiBase + "/api/v1/auth/refreshtoken")
	if err != nil {
		return err
	}
	if res.StatusCode() >= 400 {
		return fmt.Errorf("failed to refresh token: %s", res.String())
	}
	if resp.IDToken == "" || resp.UID == "" {
		return fmt.Errorf("failed to refresh token: empty token response")
	}
	d.IDToken = resp.IDToken
	d.UID = resp.UID
	if resp.RefreshToken != "" {
		d.RefreshToken = resp.RefreshToken
	}
	op.MustSaveDriverStorage(d)
	return nil
}

func (d *RakutenDrive) request(pathname string, method string, callback base.ReqCallback, resp interface{}, retry ...bool) ([]byte, error) {
	req := base.RestyClient.R()
	req.SetHeaders(map[string]string{
		"Accept":          "application/json, text/plain, */*",
		"Authorization":   "Bearer " + d.IDToken,
		"Content-Type":    "application/json",
		"Origin":          "https://www.rakuten-drive.com",
		"Referer":         "https://www.rakuten-drive.com/",
		"User-Agent":      base.UserAgent,
		"X-Frame-Options": "sameorigin",
	})
	if callback != nil {
		callback(req)
	}
	if resp != nil {
		req.SetResult(resp)
	}
	var e ErrResp
	req.SetError(&e)
	res, err := req.Execute(method, apiBase+pathname)
	if err != nil {
		return nil, err
	}
	if res.StatusCode() == http.StatusUnauthorized || res.StatusCode() == http.StatusForbidden {
		isRetry := len(retry) > 0 && retry[0]
		if !isRetry {
			if err := d.refreshToken(); err != nil {
				return nil, err
			}
			return d.request(pathname, method, callback, resp, true)
		}
	}
	if res.StatusCode() >= 400 {
		if e.Message != "" {
			return nil, errors.New(e.Message)
		}
		if e.Error != "" {
			return nil, errors.New(e.Error)
		}
		return nil, fmt.Errorf("rakuten drive request failed: %s", res.String())
	}
	return res.Body(), nil
}

func (d *RakutenDrive) getFiles(parent string) ([]File, error) {
	const limit = 100
	parent = normalizeDirPath(parent)
	files := make([]File, 0)
	for from := 0; ; from += limit {
		var resp FilesResp
		_, err := d.request("/cloud/service/file/v1/files", http.MethodPost, func(req *resty.Request) {
			req.SetBody(base.Json{
				"host_id":        d.UID,
				"path":           parent,
				"from":           from,
				"to":             from + limit,
				"sort_type":      "path",
				"reverse":        false,
				"thumbnail_size": 130,
			})
		}, &resp)
		if err != nil {
			return nil, err
		}
		for i := range resp.File {
			resp.File[i].parentPath = parent
			resp.File[i].filePath = resolveFilePath(parent, resp.File[i].Path, resp.File[i].IsFolder)
			resp.File[i].lastModifiedRaw = formatDeleteTime(resp.File[i].LastModified)
		}
		files = append(files, resp.File...)
		if resp.LastPage || len(resp.File) == 0 {
			break
		}
	}
	return files, nil
}

func (d *RakutenDrive) waitTask(key string) error {
	if key == "" {
		return nil
	}
	for i := 0; i < 30; i++ {
		var resp CheckResp
		_, err := d.request("/cloud/service/file/v3/files/check", http.MethodPost, func(req *resty.Request) {
			req.SetBody(base.Json{"key": key})
		}, &resp)
		if err != nil {
			return err
		}
		switch strings.ToLower(resp.State) {
		case "complete", "completed", "done", "success":
			return nil
		case "failed", "fail", "error":
			return fmt.Errorf("rakuten drive task failed: %s", resp.State)
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("rakuten drive task timeout: %s", key)
}
