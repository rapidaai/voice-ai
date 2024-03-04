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
	provider_models "github.com/lexatic/web-backend/pkg/providers"
	integration_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"golang.org/x/sync/errgroup"
)

type imageResponseProcessor struct {
	cfg               *config.AppConfig
	logger            commons.Logger
	s3Client          *s3.S3
	integrationClient clients.IntegrationServiceClient
}

type uploadRef struct {
	Key       string
	ImageType string
	Data      string
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
		irp.logger.Errorf("error while processing the image llm request %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assitant",
		}
	} else {
		return irp.unmarshalGenerateTextToImageResponse(ctx, res, cr)
	}
}

func (irp *imageResponseProcessor) unmarshalGenerateTextToImageResponse(ctx context.Context, res *integration_api.GenerateTextToImageResponse, cr *clients_pogos.RequestData[string]) *clients_pogos.PromptResponse {
	if !res.Success {
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     res.ErrorMessage,
			ResponseRole: "assitant",
			RequestId:    res.RequestId,
		}
	}

	switch providerName := strings.ToLower(cr.ProviderName); providerName {
	case "openai":
		return irp.unmarshalOpenAiImage(ctx, res)
	case "stabilityai":
		return irp.unmarshalStabilityAiImage(ctx, res)
	case "togetherai":
		return irp.unmarshalTogetherAiImage(ctx, res)
	case "deepinfra":
		return irp.unmarshalDeepInfraImage(ctx, res, cr)
	default:
		return irp.unmarshalOpenAiImage(ctx, res)
	}
}

func (irp *imageResponseProcessor) uploadReference(ctx context.Context, key, imageType, image string) error {
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
		_, err = irp.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
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
		_, err = irp.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
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

func generateDeepInfraImageRefs(res *integration_api.GenerateTextToImageResponse, cr *clients_pogos.RequestData[string]) ([]*uploadRef, error) {
	// Add utility for deep infra later , to determine what to response format to use
	if provider_models.IsDeepInfraV2ImageModel(cr.ProviderModelName) {
		deepInfraRes := clients_pogos.DeepInfraImageResponse{}
		err := json.Unmarshal([]byte(*res.Response), &deepInfraRes)
		if err != nil {
			return nil, err
		}
		refs := make([]*uploadRef, len(deepInfraRes.Output))
		for i, output := range deepInfraRes.Output {
			key := fmt.Sprintf("output/image/%d_%s.png", res.RequestId, ciphers.RandomHash("img_"))
			refs[i] = &uploadRef{
				Key:       key,
				ImageType: "url",
				Data:      output,
			}
		}
		return refs, nil
	}

	deepInfraLegacyRes := clients_pogos.DeepInfraImageLegacyResponse{}
	err := json.Unmarshal([]byte(*res.Response), &deepInfraLegacyRes)
	if err != nil {
		return nil, err
	}
	refs := make([]*uploadRef, len(deepInfraLegacyRes.Images))
	for i, image := range deepInfraLegacyRes.Images {
		key := fmt.Sprintf("output/image/%d_%s.png", res.RequestId, ciphers.RandomHash("img_"))
		refs[i] = &uploadRef{
			Key:       key,
			ImageType: "base64",
			Data:      strings.Replace(image, "data:image/png;base64,", "", 1),
		}
	}
	return refs, nil
}

func (irp *imageResponseProcessor) unmarshalDeepInfraImage(ctx context.Context, res *integration_api.GenerateTextToImageResponse, cr *clients_pogos.RequestData[string]) *clients_pogos.PromptResponse {
	refs, err := generateDeepInfraImageRefs(res, cr)
	if err != nil {
		irp.logger.Errorf("unmarshalDeepInfraImage error %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assistant",
		}
	}
	responses, err := irp.uploadReferences(ctx, refs)
	if err != nil {
		irp.logger.Errorf("unable to upload responses error %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assitant",
		}
	}

	jsonString, err := json.Marshal(responses)
	if err != nil {
		irp.logger.Errorf("unmarshalDeepInfraImage error %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assistant",
		}
	}

	return &clients_pogos.PromptResponse{
		Status:       "SUCCESS",
		ResponseRole: "system",
		Response:     string(jsonString),
		RequestId:    res.RequestId,
	}
}

func (irp *imageResponseProcessor) unmarshalTogetherAiImage(ctx context.Context, res *integration_api.GenerateTextToImageResponse) *clients_pogos.PromptResponse {
	// already checked for success
	togetherAIRes := clients_pogos.TogetherAIResponse[clients_pogos.TogetherAiImageChoice]{}
	err := json.Unmarshal([]byte(*res.Response), &togetherAIRes)
	if err != nil {
		irp.logger.Errorf("unmarshalTogetherAiImage error %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assistant",
		}
	}
	refs := make([]*uploadRef, len(togetherAIRes.Choices))
	for i, choice := range togetherAIRes.Choices {
		key := fmt.Sprintf("output/image/%d_%s.png", res.RequestId, ciphers.RandomHash("img_"))
		refs[i] = &uploadRef{
			Key:       key,
			ImageType: "base64",
			Data:      choice.ImageBase64,
		}
	}
	responses, err := irp.uploadReferences(ctx, refs)
	if err != nil {
		irp.logger.Errorf("unable to upload responses error %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assitant",
		}
	}

	jsonString, err := json.Marshal(responses)
	if err != nil {
		irp.logger.Errorf("unmarshalTogetherAiImage error %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assistant",
		}
	}

	return &clients_pogos.PromptResponse{
		Status:       "SUCCESS",
		ResponseRole: "system",
		Response:     string(jsonString),
		RequestId:    res.RequestId,
	}
}

func (irp *imageResponseProcessor) unmarshalStabilityAiImage(ctx context.Context, res *integration_api.GenerateTextToImageResponse) *clients_pogos.PromptResponse {
	if res.Success {
		stabilityRes := clients_pogos.StabilityAIImageResponse{}
		err := json.Unmarshal([]byte(*res.Response), &stabilityRes)
		if err != nil {
			irp.logger.Errorf("unmarshalStabilityAiImage error %v", err)
			return &clients_pogos.PromptResponse{
				Status:       "FAILURE",
				Response:     err.Error(),
				ResponseRole: "assistant",
			}
		}
		refs := make([]*uploadRef, len(stabilityRes.Artifacts))
		for i, artifact := range stabilityRes.Artifacts {
			key := fmt.Sprintf("output/image/%d_%s.png", res.RequestId, ciphers.RandomHash("img_"))
			refs[i] = &uploadRef{
				Key:       key,
				ImageType: "base64",
				Data:      artifact.Base64,
			}
		}
		responses, err := irp.uploadReferences(ctx, refs)

		if err != nil {
			irp.logger.Errorf("unable to upload responses error %v", err)
			return &clients_pogos.PromptResponse{
				Status:       "FAILURE",
				Response:     err.Error(),
				ResponseRole: "assitant",
			}
		}

		jsonString, err := json.Marshal(responses)
		if err != nil {
			irp.logger.Errorf("unmarshalStabilityAiImage error %v", err)
			return &clients_pogos.PromptResponse{
				Status:       "FAILURE",
				Response:     err.Error(),
				ResponseRole: "assistant",
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
			Response:     res.ErrorMessage,
			ResponseRole: "assitant",
			RequestId:    res.RequestId,
		}
	}
}

func (irp *imageResponseProcessor) uploadReferences(ctx context.Context, refs []*uploadRef) ([]string, error) {
	group, qctx := errgroup.WithContext(ctx)
	responses := make([]string, len(refs))

	for i, ref := range refs {
		func(k, imageType, iD string, index int) {
			group.Go(func() error {
				err := irp.uploadReference(qctx, k, imageType, iD)
				responses[index] = k
				return err
			})
		}(ref.Key, ref.ImageType, ref.Data, i)
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return responses, nil
}

func (irp *imageResponseProcessor) unmarshalOpenAiImage(ctx context.Context, res *integration_api.GenerateTextToImageResponse) *clients_pogos.PromptResponse {
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

		refs := make([]*uploadRef, len(openAiRes.Data))
		for i, img := range openAiRes.Data {
			key := fmt.Sprintf("output/image/%d_%s.png", res.RequestId, ciphers.RandomHash("img_"))
			if bs64, url := img.B64Json, img.Url; bs64 != nil {
				refs[i] = &uploadRef{
					Key:       key,
					ImageType: "base64",
					Data:      *bs64,
				}
			} else {
				refs[i] = &uploadRef{
					Key:       key,
					ImageType: "url",
					Data:      *url,
				}
			}
		}

		responses, err := irp.uploadReferences(ctx, refs)

		if err != nil {
			irp.logger.Errorf("unmarshalOpenAiImage error %v", err)
			return &clients_pogos.PromptResponse{
				Status:       "FAILURE",
				Response:     err.Error(),
				ResponseRole: "assitant",
			}
		}

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
			Response:     res.ErrorMessage,
			ResponseRole: "assitant",
			RequestId:    res.RequestId,
		}
	}
}
