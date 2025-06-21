package v1

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/chindada/capitan/internal/controller/http/resp"
	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/launcher"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type systemRoutes struct{}

func NewSystemRoutes(handler *gin.RouterGroup) {
	r := &systemRoutes{}
	base := "/system"

	h := handler.Group(base)
	{
		h.GET("/backup", r.listBackup)
		h.PUT("/backup", r.createBackup)
		h.POST("/backup", r.restoreBackup)
		h.DELETE("/backup", r.deleteBackup)
		h.GET("/backup/download", r.downloadBackup)
		h.POST("/backup/upload", r.uploadBackup)
	}
}

// createBackup -.
//
//	@Tags		System V1
//	@Summary	Create backup
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	emptypb.Empty
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/system/backup [put]
func (r *systemRoutes) createBackup(c *gin.Context) {
	dbt := launcher.Get()
	err := dbt.Backup(false)
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}

// restoreBackup -.
//
//	@Tags		System V1
//	@Summary	Restore backup
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		body	body		pb.Backup	true	"Body"
//	@Success	200		{object}	emptypb.Empty
//	@Failure	400		{object}	pb.APIResponse
//	@Failure	500		{object}	pb.APIResponse
//	@Router		/api/capitan/v1/system/backup [post]
func (r *systemRoutes) restoreBackup(c *gin.Context) {
	backup := &pb.Backup{}
	err := c.Bind(backup)
	if err != nil {
		resp.Fail(c, http.StatusBadRequest, err)
		return
	}
	dbt := launcher.Get()
	if err = dbt.RestoreDatabase(backup.GetName()); err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	p, e := os.FindProcess(os.Getpid())
	if e != nil {
		resp.Fail(c, http.StatusInternalServerError, e)
		return
	}
	if runtime.GOOS == "windows" {
		e = p.Kill()
		if e != nil {
			resp.Fail(c, http.StatusInternalServerError, e)
			return
		}
	} else {
		e = p.Signal(syscall.SIGQUIT)
		if e != nil {
			resp.Fail(c, http.StatusInternalServerError, e)
			return
		}
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}

// listBackup -.
//
//	@Tags		System V1
//	@Summary	List backup
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@Success	200	{object}	pb.BackupList
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/system/backup [get]
func (r *systemRoutes) listBackup(c *gin.Context) {
	dbt := launcher.Get()
	list, err := dbt.ListBackups()
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	res := []*pb.Backup{}
	for _, v := range list {
		res = append(res, &pb.Backup{
			Name:      v.Name,
			Path:      v.Path,
			CreatedAt: timestamppb.New(v.CreatedAt),
		})
	}
	resp.Success(c, http.StatusOK, &pb.BackupList{
		List: res,
	})
}

// downloadBackup -.
//
//	@Tags		System V1
//	@Summary	Download backup
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		backup-name	header		string	true	"backup-name"
//	@Failure	400			{object}	pb.APIResponse
//	@Failure	500			{object}	pb.APIResponse
//	@Router		/api/capitan/v1/system/backup/download [get]
func (r *systemRoutes) downloadBackup(c *gin.Context) {
	backupName := c.GetHeader("backup-name")
	if backupName == "" {
		resp.Fail(c, http.StatusBadRequest, resp.ErrNameRequired)
		return
	}

	dbt := launcher.Get()
	list, err := dbt.ListBackups()
	if err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	var filePath string
	for _, v := range list {
		if v.Name == backupName {
			filePath = v.Path
			break
		}
	}
	if filePath == "" {
		resp.Fail(c, http.StatusBadRequest, resp.ErrNotFound)
		return
	}
	zipName := fmt.Sprintf("%s.zip", filepath.Base(filePath))
	zipPath := filepath.Join(filepath.Dir(filePath), zipName)
	if err = dbt.Zip(zipPath, filePath); err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	c.FileAttachment(
		zipPath,
		zipName,
	)
	_ = os.Remove(zipPath)
}

// uploadBackup -.
//
//	@Tags		System V1
//	@Summary	Upload backup
//	@security	JWT
//	@accept		multipart/form-data
//	@param		file	formData	file	true	"file"
//	@Produce	application/json
//	@Success	200	{object}	emptypb.Empty
//	@Failure	500	{object}	pb.APIResponse
//	@Router		/api/capitan/v1/system/backup/upload [post]
func (r *systemRoutes) uploadBackup(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		return
	}
	savePath := filepath.Join(os.TempDir(), file.Filename)
	_ = c.SaveUploadedFile(file, savePath)
	dbt := launcher.Get()
	if err = dbt.LoadBackupArchiveFile(savePath); err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, &emptypb.Empty{})
}

// deleteBackup -.
//
//	@Tags		System V1
//	@Summary	Delete backup
//	@security	JWT
//	@Accept		application/json
//	@Produce	application/json
//	@param		backup-name	header		string	true	"backup-name"
//	@Success	200			{object}	emptypb.Empty
//	@Failure	400			{object}	pb.APIResponse
//	@Failure	500			{object}	pb.APIResponse
//	@Router		/api/capitan/v1/system/backup [delete]
func (r *systemRoutes) deleteBackup(c *gin.Context) {
	backupName := c.GetHeader("backup-name")
	if backupName == "" {
		resp.Fail(c, http.StatusBadRequest, resp.ErrNameRequired)
		return
	}
	dbt := launcher.Get()
	if err := dbt.DeleteBackup(backupName); err != nil {
		resp.Fail(c, http.StatusInternalServerError, err)
		return
	}
	resp.Success(c, http.StatusOK, &emptypb.Empty{})
}
