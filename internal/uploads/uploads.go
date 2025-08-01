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
	RemoveImgFlag      = "delete"
	MaxImageUploadSize = 64 << 20
)

var (
	ErrFileHeaderOpenFail = errors.New("failed to open file from fileheader")
	ErrFileCreateFail     = errors.New("failed to create output file")
	ErrFileCopyFail       = errors.New("failed to copy file")

	UploadsPath = "web/static/uploads"
)

type WrittenFile struct {
	Filename string
	Size     int64
}

func writeFile(file *multipart.FileHeader, writePath string) (int64, error) {
	p, err := file.Open()
	if err != nil {
		return 0, errors.Join(ErrFileHeaderOpenFail, err)
	}
	defer p.Close()

	outFile, err := os.Create(writePath)
	if err != nil {
		return 0, errors.Join(ErrFileCreateFail, err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, p)
	if err != nil {
		return 0, errors.Join(ErrFileCopyFail, err)
	}

	return written, nil
}

func Upload(file *multipart.FileHeader) (filename string, err error) {
	date := time.Now().Format("2006-01-02T15:04:05")
	uploadsPath := UploadsPath

	filename = fmt.Sprintf("upload_%s_%d%s", date, 0, filepath.Ext(file.Filename))
	writePath := filepath.Join(uploadsPath, filename)
	_, err = writeFile(file, writePath)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func UploadMultiple(files []*multipart.FileHeader) (writtenFiles []*WrittenFile, err error) {
	date := time.Now().Format("2006-01-02T15:04:05")
	uploadsPath := UploadsPath
	writtenFiles = make([]*WrittenFile, len(files))

	for i, fHeader := range files {
		filename := fmt.Sprintf("upload_%s_%d%s", date, i, filepath.Ext(fHeader.Filename))
		writePath := filepath.Join(uploadsPath, filename)

		sz, err := writeFile(fHeader, writePath)
		if err != nil {
			return nil, err
		}

		writtenFiles[i] = &WrittenFile{
			Filename: filename,
			Size:     sz,
		}
	}

	return writtenFiles, nil
}

func Update(filename string, newFile *multipart.FileHeader) error {
	writePath := filepath.Join(UploadsPath, filename)
	_, err := writeFile(newFile, writePath)
	if err != nil {
		return err
	}

	return nil
}

func Delete(filename string) error {
	delPath := filepath.Join(UploadsPath, filename)

	return os.Remove(delPath)
}

func DeleteMultiple(filenames []string) error {
	for _, filename := range filenames {
		delPath := filepath.Join(UploadsPath, filename)
		err := os.Remove(delPath)
		if err != nil {
			return err
		}
	}

	return nil
}
