package clients_response_processors

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lexatic/web-backend/config"
	"github.com/lexatic/web-backend/pkg/ciphers"
	clients "github.com/lexatic/web-backend/pkg/clients"
	integration_service_client "github.com/lexatic/web-backend/pkg/clients/integration"
	clients_pogos "github.com/lexatic/web-backend/pkg/clients/pogos"
	"github.com/lexatic/web-backend/pkg/commons"
	integration_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type imageResponseProcessor struct {
	cfg               *config.AppConfig
	logger            commons.Logger
	s3Client          *s3.S3
	integrationClient clients.IntegrationServiceClient
}

func NewImageResponseProcessor(cfg *config.AppConfig, lgr commons.Logger) ResponseProcessor[string] {
	config := aws.Config{
		Region: aws.String(cfg.AssetStoreConfig.Auth.Region),
	}
	if cfg.AssetStoreConfig.Auth.AccessKeyId != "" && cfg.AssetStoreConfig.Auth.SecretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(
			cfg.AssetStoreConfig.Auth.AccessKeyId,
			cfg.AssetStoreConfig.Auth.SecretKey,
			"",
		)
	}
	sessionOptions := awsSession.Options{
		Config:            config,
		SharedConfigState: awsSession.SharedConfigEnable,
	}

	_session, err := awsSession.NewSessionWithOptions(sessionOptions)
	if err != nil {
		lgr.Errorf("unable to download the dataset files with error %v", err)
		return &imageResponseProcessor{logger: lgr, cfg: cfg, integrationClient: integration_service_client.NewIntegrationServiceClientGRPC(cfg, lgr)}
	}

	return &imageResponseProcessor{logger: lgr, cfg: cfg, integrationClient: integration_service_client.NewIntegrationServiceClientGRPC(cfg, lgr), s3Client: s3.New(_session)}
}

func (irp *imageResponseProcessor) Process(ctx context.Context, cr *clients_pogos.RequestData[string]) *clients_pogos.PromptResponse {
	if res, err := irp.integrationClient.GenerateTextToImage(ctx, cr); err != nil {
		irp.logger.Info("Unable to launch req %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assitant",
		}
	} else {
		return irp.unmarshalGenerateTextToImageResponse(res, cr.ProviderName)
	}
}

func (irp *imageResponseProcessor) unmarshalGenerateTextToImageResponse(res *integration_api.GenerateTextToImageResponse, provider string) *clients_pogos.PromptResponse {
	switch providerName := strings.ToLower(provider); providerName {
	case "openai":
		return irp.unmarshalOpenAiImage(res)
	default:
		return irp.unmarshalOpenAiImage(res)
	}
}

func (irp *imageResponseProcessor) uploadReference(key string, imageType string, image string) error {
	// key := fmt.Sprintf("%d/%d/response.png", experimentId, requestId)
	switch imageType {
	case "url":
		// Fetch image from URL
		resp, err := http.Get(image)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unable to fetch image from URL: %s", resp.Status)
		}
		// Read image data into a buffer
		imageData, err := io.ReadAll(resp.Body)
		if err != nil {
			irp.logger.Errorf("error while reading image data err %v", err)
			return err
		}
		// Upload image data to S3
		_, err = irp.s3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(irp.cfg.AssetStoreConfig.AssetUploadBucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(imageData),
		})
		if err != nil {
			irp.logger.Errorf("error while uploading image to s3 err %v", err)
			return err
		}
	case "base64":
		// Decode base64 string to image data
		decoded, err := base64.StdEncoding.DecodeString(image)
		if err != nil {
			irp.logger.Errorf("error while reading image data err %v", err)
			return err
		}
		// Upload decoded image data to S3
		_, err = irp.s3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(irp.cfg.AssetStoreConfig.AssetUploadBucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(decoded),
		})
		if err != nil {
			irp.logger.Errorf("error while uploading image to s3 err %v", err)
			return err
		}
	default:
		return fmt.Errorf("unsupported image type: %s", imageType)
	}

	return nil
}

func (irp *imageResponseProcessor) unmarshalOpenAiImage(res *integration_api.GenerateTextToImageResponse) *clients_pogos.PromptResponse {
	if res.Success {
		openAiRes := clients_pogos.OpenAIImageResponse{}
		err := json.Unmarshal([]byte(*res.Response), &openAiRes)
		if err != nil {
			irp.logger.Errorf("unmarshalOpenAiImage error %v", err)
			return &clients_pogos.PromptResponse{
				Status:       "FAILURE",
				Response:     err.Error(),
				ResponseRole: "assitant",
			}
		}
		responses := make([]string, 0)
		var wg sync.WaitGroup
		for _, img := range openAiRes.Data {
			key := fmt.Sprintf("output/image/%d_%s.png", res.RequestId, ciphers.RandomHash("img_"))
			irp.logger.Debugf("uploading assets to s3 %v", key)
			wg.Add(1)
			if bs64, url := img.B64Json, img.Url; bs64 != nil {
				go func(k, imageType string, iD string) {
					defer wg.Done()
					// Fetch the URL.
					irp.uploadReference(k, imageType, iD)
				}(key, "base64", *bs64)
			} else {
				go func(k, imageType string, iD string) {
					defer wg.Done()
					// Fetch the URL.
					irp.uploadReference(k, imageType, iD)
				}(key, "url", *url)
			}
			responses = append(responses, key)
		}
		// Wait for all HTTP fetches to complete.

		jsonString, err := json.Marshal(responses)
		if err != nil {
			irp.logger.Errorf("unmarshalOpenAiImage error %v", err)
			return &clients_pogos.PromptResponse{
				Status:       "FAILURE",
				Response:     err.Error(),
				ResponseRole: "assitant",
			}
		}

		return &clients_pogos.PromptResponse{
			Status:       "SUCCESS",
			ResponseRole: "system",
			Response:     string(jsonString),
			RequestId:    res.RequestId,
		}

	} else {
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     *res.Response,
			ResponseRole: "assitant",
			RequestId:    res.RequestId,
		}
	}
}
