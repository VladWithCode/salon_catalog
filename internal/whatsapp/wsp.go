package whatsapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

type TemplateVar map[string]any

type TemplateComponent struct {
	ComponentType string        `json:"type"`
	Parameters    []TemplateVar `json:"parameters"`
}

type TemplateData struct {
	TemplateName string
	BodyVars     []TemplateVar
	HeaderVars   []TemplateVar
}

type templatePayload struct {
	Name       string              `json:"name"`
	Components []TemplateComponent `json:"components,omitempty"`
	Language   struct {
		Code string `json:"code"`
	} `json:"language"`
}

var (
	ErrPhoneNotSet         = errors.New("phone number is not set in environment variables")
	ErrEnvVarsNotSet       = errors.New("environment variables for whatsapp messaging are not set")
	ErrPayloadInvalid      = errors.New("request payload is invalid")
	ErrRequestFailed       = errors.New("request to facebook's api failed")
	ErrRequestCreateFailed = errors.New("failed to create http.Request object")
)

const (
	TemplateNameQuoteRequest = "customer_quote_confirmation"
	EnvVarPhoneNumberId      = "WSP_PHONE_NUMBER_ID"
	EnvVarAccessToken        = "WSP_ACCESS_TOKEN"
	EnvVarUseVersion         = "WSP_USE_VERSION"
)

func postCloudAPIMessage(requestPayload any) error {
	phoneNumberId := os.Getenv(EnvVarPhoneNumberId)
	fbAccessToken := os.Getenv(EnvVarAccessToken)
	useVersion := os.Getenv(EnvVarUseVersion)

	if phoneNumberId == "" || fbAccessToken == "" {
		return ErrEnvVarsNotSet
	}
	if useVersion == "" {
		useVersion = "v22.0"
	}

	reqBody, err := json.Marshal(requestPayload)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPayloadInvalid, err)
	}

	reqUrl := fmt.Sprintf("https://graph.facebook.com/%s/%v/messages", useVersion, phoneNumberId)
	req, err := http.NewRequest("post", reqUrl, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRequestCreateFailed, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", fbAccessToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		data, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Printf("failed to read response body: %v\n", err)
		}
		dataStr := string(data)
		return fmt.Errorf("request failed with code: %v, body: %v", resp.StatusCode, dataStr)
	}

	return nil
}

func SendTemplateMessage(phoneNumber string, data TemplateData) error {
	var reqPayload struct {
		MessagingProduct string          `json:"messaging_product"`
		MessageType      string          `json:"type"`
		ToPhone          string          `json:"to"`
		Template         templatePayload `json:"template"`
	}

	var components []TemplateComponent

	if len(data.HeaderVars) != 0 {
		components = append(components, TemplateComponent{
			ComponentType: "header",
			Parameters:    data.HeaderVars,
		})
	}

	if len(data.BodyVars) != 0 {
		components = append(components, TemplateComponent{
			ComponentType: "body",
			Parameters:    data.BodyVars,
		})
	}

	reqPayload.MessageType = "template"
	reqPayload.MessagingProduct = "whatsapp"
	reqPayload.ToPhone = phoneNumber
	reqPayload.Template = templatePayload{
		Name: data.TemplateName,
		Language: struct {
			Code string `json:"code"`
		}{Code: "es"},
		Components: components,
	}

	return postCloudAPIMessage(reqPayload)
}

func SendQuoteRequestConfirmation(quoteData *db.Quote) error {
	sendTo := quoteData.CustomerPhone
	if sendTo == "" {
		return ErrPhoneNotSet
	}

	return SendTemplateMessage(sendTo, TemplateData{
		TemplateName: TemplateNameQuoteRequest,
	})
}
