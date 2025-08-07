package qrgen

import (
	"fmt"
	"path"

	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type QRCodeData struct {
	Filename string `json:"filename"`
	Value    string `json:"value"`
}

func GenerateFromString(qrdata *QRCodeData) (string, error) {
	qrc, err := qrcode.New(qrdata.Value)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("web/static/qrcodes/%s.jpeg", qrdata.Filename)
	w, err := standard.New(filename)
	if err != nil {
		return "", err
	}

	err = qrc.Save(w)
	if err != nil {
		return "", err
	}

	return path.Base(filename), nil
}
