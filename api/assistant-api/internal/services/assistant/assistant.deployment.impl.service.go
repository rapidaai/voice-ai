package internal_assistant_service

import (
	"context"
	"errors"

	"github.com/rapidaai/api/assistant-api/config"
	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	lexatic_backend "github.com/rapidaai/protos"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type assistantDeploymentService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	cfg      *config.AssistantConfig
}

func NewAssistantDeploymentService(cfg *config.AssistantConfig,
	logger commons.Logger,
	postgres connectors.PostgresConnector) internal_services.AssistantDeploymentService {
	return &assistantDeploymentService{
		logger:   logger,
		postgres: postgres,
		cfg:      cfg,
	}
}

func (eService assistantDeploymentService) CreateWebPluginDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	name, icon string,
	greeting, mistake *string,
	suggestion []string,
	helpCenterEnabled, productCatalogEnabled, articleCatalogEnabled bool,
	inputAudio, outputAudio *lexatic_backend.DeploymentAudioProvider,
) (*internal_assistant_gorm.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_gorm.AssistantWebPluginDeployment{
		AssistantDeploymentBehavior: internal_assistant_gorm.AssistantDeploymentBehavior{
			AssistantDeploymentPersona: internal_assistant_gorm.AssistantDeploymentPersona{
				AssistantDeployment: internal_assistant_gorm.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
					},
					AssistantId: assistantId,
				},
				Name: name,
			},
			Greeting: greeting,
			Mistake:  mistake,
		},
		Icon:                  icon,
		Suggestion:            suggestion,
		HelpCenterEnabled:     helpCenterEnabled,
		ProductCatalogEnabled: productCatalogEnabled,
		ArticleCatalogEnabled: articleCatalogEnabled,
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	//
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) createAssistantDeploymentAudio(
	ctx context.Context,
	auth types.SimplePrinciple, deploymentId uint64,
	audioType string,
	audioConfig *lexatic_backend.DeploymentAudioProvider) (*internal_assistant_gorm.AssistantDeploymentAudio, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_gorm.AssistantDeploymentAudio{
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			Status:    type_enums.RecordState(audioConfig.GetStatus()),
		},
		AudioType:             audioType,
		AssistantDeploymentId: deploymentId,
		AudioProviderId:       audioConfig.GetAudioProviderId(),
		AudioProvider:         audioConfig.GetAudioProvider(),
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create deployment audio config for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if len(audioConfig.GetAudioOptions()) == 0 {
		return deployment, nil
	}
	audioDeploymentOptions := make([]*internal_assistant_gorm.AssistantDeploymentAudioOption, 0)
	for _, v := range audioConfig.GetAudioOptions() {
		audioDeploymentOptions = append(audioDeploymentOptions, &internal_assistant_gorm.AssistantDeploymentAudioOption{
			AssistantDeploymentAudioId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RecordState(audioConfig.GetStatus()),
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_audio_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(audioDeploymentOptions)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create deployment audio config metadata for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	return deployment, nil
}

func (eService assistantDeploymentService) CreateDebuggerDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	name, icon string,
	greeting, mistake *string,
	inputAudio, outputAudio *lexatic_backend.DeploymentAudioProvider,
) (*internal_assistant_gorm.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_gorm.AssistantDebuggerDeployment{

		AssistantDeploymentBehavior: internal_assistant_gorm.AssistantDeploymentBehavior{
			AssistantDeploymentPersona: internal_assistant_gorm.AssistantDeploymentPersona{
				AssistantDeployment: internal_assistant_gorm.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_ACTIVE,
					},
					AssistantId: assistantId,
				},
				Name: name,
			},
			Greeting: greeting,
			Mistake:  mistake,
		},
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) CreateApiDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	inputAudio, outputAudio *lexatic_backend.DeploymentAudioProvider,
) (*internal_assistant_gorm.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_gorm.AssistantApiDeployment{
		AssistantDeploymentBehavior: internal_assistant_gorm.AssistantDeploymentBehavior{
			AssistantDeploymentPersona: internal_assistant_gorm.AssistantDeploymentPersona{
				AssistantDeployment: internal_assistant_gorm.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_ACTIVE,
					},
					AssistantId: assistantId,
				},
			},
			Greeting: greeting,
			Mistake:  mistake,
		},
	}

	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	return deployment, nil
}

func (eService assistantDeploymentService) CreateWhatsappDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	whatsappProviderId uint64, whatsappProvider string,
	whatsappOptions []*lexatic_backend.Metadata,
) (*internal_assistant_gorm.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_gorm.AssistantWhatsappDeployment{
		AssistantDeploymentBehavior: internal_assistant_gorm.AssistantDeploymentBehavior{
			AssistantDeploymentPersona: internal_assistant_gorm.AssistantDeploymentPersona{
				AssistantDeployment: internal_assistant_gorm.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_ACTIVE,
					},
					AssistantId: assistantId,
				},
			},
			Greeting: greeting,
			Mistake:  mistake,
		},
		AssistantDeploymentWhatsapp: internal_assistant_gorm.AssistantDeploymentWhatsapp{
			WhatsappProviderId: whatsappProviderId,
			WhatsappProvider:   whatsappProvider,
		},
	}

	// TODO: Persist the deployment to the database
	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if len(whatsappOptions) == 0 {
		return deployment, nil
	}

	whatsappOpts := make([]*internal_assistant_gorm.AssistantDeploymentWhatsappOption, 0)
	for _, v := range whatsappOptions {
		whatsappOpts = append(whatsappOpts, &internal_assistant_gorm.AssistantDeploymentWhatsappOption{
			AssistantDeploymentWhatsappId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RECORD_ACTIVE,
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_whatsapp_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(whatsappOpts)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create whatsapp options for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}
	return deployment, nil
}

func (eService assistantDeploymentService) CreatePhoneDeployment(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	greeting, mistake *string,
	phoneProviderId uint64, phoneProvider string,
	inputAudio, outputAudio *lexatic_backend.DeploymentAudioProvider,
	opts []*lexatic_backend.Metadata,
) (*internal_assistant_gorm.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_gorm.AssistantPhoneDeployment{
		AssistantDeploymentBehavior: internal_assistant_gorm.AssistantDeploymentBehavior{
			AssistantDeploymentPersona: internal_assistant_gorm.AssistantDeploymentPersona{
				AssistantDeployment: internal_assistant_gorm.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_ACTIVE,
					},
					AssistantId: assistantId,
				},
			},
			Greeting: greeting,
			Mistake:  mistake,
		},
		AssistantDeploymentTelephony: internal_assistant_gorm.AssistantDeploymentTelephony{
			TelephonyProviderId: phoneProviderId,
			TelephonyProvider:   phoneProvider,
		},
	}

	// TODO: Persist the deployment to the database
	tx := db.Create(deployment)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create web plugin deployment for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	if inputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "input", inputAudio)
	}
	if outputAudio != nil {
		eService.createAssistantDeploymentAudio(ctx, auth, deployment.Id, "output", outputAudio)
	}

	if len(opts) == 0 {
		eService.logger.Warnf("no options for the telephony provider.")
		return deployment, nil
	}

	phoneOpts := make([]*internal_assistant_gorm.AssistantDeploymentTelephonyOption, 0)
	for _, v := range opts {
		phoneOpts = append(phoneOpts, &internal_assistant_gorm.AssistantDeploymentTelephonyOption{
			AssistantDeploymentTelephonyId: deployment.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RECORD_ACTIVE,
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}

	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_deployment_telephony_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(phoneOpts)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create telephony options for assistant wiht error %v", tx.Error)
		return nil, tx.Error
	}

	return deployment, nil
}

func (eService assistantDeploymentService) GetAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	var apiDeployment *internal_assistant_gorm.AssistantApiDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OuputAudio", "audio_type = ?", "output").
		Preload("OuputAudio.AudioOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&apiDeployment)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return apiDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	var debuggerDeployment *internal_assistant_gorm.AssistantDebuggerDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OuputAudio", "audio_type = ?", "output").
		Preload("OuputAudio.AudioOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&debuggerDeployment)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return debuggerDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	var phoneDeployment *internal_assistant_gorm.AssistantPhoneDeployment
	qry := db.
		Preload("TelephonyOption").
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OuputAudio", "audio_type = ?", "output").
		Preload("OuputAudio.AudioOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&phoneDeployment)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return phoneDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	var webPluginDeployment *internal_assistant_gorm.AssistantWebPluginDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OuputAudio", "audio_type = ?", "output").
		Preload("OuputAudio.AudioOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&webPluginDeployment)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("not able to find web plugin deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return webPluginDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	var whatsappDeployment *internal_assistant_gorm.AssistantWhatsappDeployment
	qry := db.
		Preload("WhatsappOptions").
		Where("assistant_id = ?", assistantId)
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&whatsappDeployment)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("not able to find whatsapp deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return whatsappDeployment, nil
}
