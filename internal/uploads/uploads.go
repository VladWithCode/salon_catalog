// Package uploads provides functions for uploading files
package uploads

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

const (
	RemoveImgFlag = "delete"
)

var (
	ErrFileHeaderOpenFail = errors.New("failed to open file from fileheader")
	ErrFileCreateFail     = errors.New("failed to create output file")
	ErrFileCopyFail       = errors.New("failed to copy file")

	UploadsPath = "web/static/uploads"
)

func writeFile(file *multipart.FileHeader, writePath string) error {
	p, err := file.Open()
	if err != nil {
		return errors.Join(ErrFileHeaderOpenFail, err)
	}
	defer p.Close()

	outFile, err := os.Create(writePath)
	if err != nil {
		return errors.Join(ErrFileCreateFail, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, p)
	if err != nil {
		return errors.Join(ErrFileCopyFail, err)
	}

	return nil
}

func Upload(file *multipart.FileHeader) (filename string, err error) {
	date := time.Now().Format("2006-01-02T15:04:05")
	uploadsPath := UploadsPath

	filename = fmt.Sprintf("upload_%s_%d%s", date, 0, filepath.Ext(file.Filename))
	writePath := filepath.Join(uploadsPath, filename)
	err = writeFile(file, writePath)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func UploadMultiple(files []*multipart.FileHeader) (fileNames []string, err error) {
	date := time.Now().Format("2006-01-02T15:04:05")
	uploadsPath := UploadsPath
	fileNames = make([]string, len(files))

	for i, fHeader := range files {
		filename := fmt.Sprintf("upload_%s_%d%s", date, i, filepath.Ext(fHeader.Filename))
		writePath := filepath.Join(uploadsPath, filename)

		err = writeFile(fHeader, writePath)
		if err != nil {
			return nil, err
		}

		fileNames[i] = filename
	}

	return fileNames, nil
}

func Update(filename string, newFile *multipart.FileHeader) error {
	writePath := filepath.Join(UploadsPath, filename)
	err := writeFile(newFile, writePath)
	if err != nil {
		return err
	}

	return nil
}

func Delete(filename string) error {
	delPath := filepath.Join(UploadsPath, filename)

	return os.Remove(delPath)
}
