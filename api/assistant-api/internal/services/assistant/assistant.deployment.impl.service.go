// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"errors"
	"fmt"

	"github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
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
	greeting, mistake *string,
	unclearInputTimeout *float64, unclearInputMessage *string,
	greetingInterruptible *bool,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	suggestion []string,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantWebPluginDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			Mistake:               mistake,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           IdleTimeout,
			IdleTimeoutBackoff:    IdleTimeoutBackoff,
			IdleTimeoutMessage:    IdleTimeoutMessage,
			MaxSessionDuration:    maxSessionDuration,
		},
		Suggestion: suggestion,
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantWebPluginDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
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
	audioConfig *protos.DeploymentAudioProvider) (*internal_assistant_entity.AssistantDeploymentAudio, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantDeploymentAudio{
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			Status:    type_enums.RecordState(audioConfig.GetStatus()),
		},
		AudioType:             audioType,
		AssistantDeploymentId: deploymentId,
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
	audioDeploymentOptions := make([]*internal_assistant_entity.AssistantDeploymentAudioOption, 0)
	for _, v := range audioConfig.GetAudioOptions() {
		audioDeploymentOptions = append(audioDeploymentOptions, &internal_assistant_entity.AssistantDeploymentAudioOption{
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
	greeting, mistake *string,
	unclearInputTimeout *float64, unclearInputMessage *string,
	greetingInterruptible *bool,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantDebuggerDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			Mistake:               mistake,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           IdleTimeout,
			IdleTimeoutBackoff:    IdleTimeoutBackoff,
			IdleTimeoutMessage:    IdleTimeoutMessage,
			MaxSessionDuration:    maxSessionDuration,
		},
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantDebuggerDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
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
	unclearInputTimeout *float64, unclearInputMessage *string,
	greetingInterruptible *bool,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantApiDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			Mistake:               mistake,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           IdleTimeout,
			IdleTimeoutBackoff:    IdleTimeoutBackoff,
			IdleTimeoutMessage:    IdleTimeoutMessage,
			MaxSessionDuration:    maxSessionDuration,
		},
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantApiDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
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
	unclearInputTimeout *float64, unclearInputMessage *string,
	greetingInterruptible *bool,
	idleTimeout *uint64,
	idleTimeoutBackoff *uint64,
	idleTimeoutMessage *string, maxSessionDuration *uint64,
	whatsappProvider string,
	whatsappOptions []*protos.Metadata,
) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantWhatsappDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			Mistake:               mistake,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           idleTimeout,
			IdleTimeoutBackoff:    idleTimeoutBackoff,
			IdleTimeoutMessage:    idleTimeoutMessage,
			MaxSessionDuration:    maxSessionDuration,
		},
		AssistantDeploymentWhatsapp: internal_assistant_entity.AssistantDeploymentWhatsapp{
			WhatsappProvider: whatsappProvider,
		},
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantWhatsappDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
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

	whatsappOpts := make([]*internal_assistant_entity.AssistantDeploymentWhatsappOption, 0)
	for _, v := range whatsappOptions {
		whatsappOpts = append(whatsappOpts, &internal_assistant_entity.AssistantDeploymentWhatsappOption{
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
	unclearInputTimeout *float64, unclearInputMessage *string,
	greetingInterruptible *bool,
	IdleTimeout *uint64,
	IdleTimeoutBackoff *uint64,
	IdleTimeoutMessage *string, maxSessionDuration *uint64,
	phoneProvider string,
	inputAudio, outputAudio *protos.DeploymentAudioProvider,
	opts []*protos.Metadata,
) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	deployment := &internal_assistant_entity.AssistantPhoneDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				Mutable: gorm_models.Mutable{
					CreatedBy: *auth.GetUserId(),
					Status:    type_enums.RECORD_ACTIVE,
				},
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			Mistake:               mistake,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           IdleTimeout,
			IdleTimeoutBackoff:    IdleTimeoutBackoff,
			IdleTimeoutMessage:    IdleTimeoutMessage,
			MaxSessionDuration:    maxSessionDuration,
		},
		AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
			TelephonyProvider: phoneProvider,
		},
	}

	if err := eService.archiveDeploymentRecords(ctx, db, &internal_assistant_entity.AssistantPhoneDeployment{}, assistantId, *auth.GetUserId()); err != nil {
		return nil, err
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

	if len(opts) == 0 {
		eService.logger.Warnf("no options for the telephony provider.")
		return deployment, nil
	}

	phoneOpts := make([]*internal_assistant_entity.AssistantDeploymentTelephonyOption, 0)
	for _, v := range opts {
		phoneOpts = append(phoneOpts, &internal_assistant_entity.AssistantDeploymentTelephonyOption{
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

func (eService assistantDeploymentService) GetAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	var apiDeployment *internal_assistant_entity.AssistantApiDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&apiDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return apiDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	var debuggerDeployment *internal_assistant_entity.AssistantDebuggerDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&debuggerDeployment)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return debuggerDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	var phoneDeployment *internal_assistant_entity.AssistantPhoneDeployment
	qry := db.
		Preload("TelephonyOption").
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&phoneDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find api deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return phoneDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	var webPluginDeployment *internal_assistant_entity.AssistantWebPluginDeployment
	qry := db.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&webPluginDeployment)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find web plugin deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return webPluginDeployment, nil
}
func (eService assistantDeploymentService) GetAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	var whatsappDeployment *internal_assistant_entity.AssistantWhatsappDeployment
	qry := db.
		Preload("WhatsappOptions").
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE})
	tx := qry.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).First(&whatsappDeployment)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if tx.Error != nil {
		eService.logger.Errorf("not able to find whatsapp deployment for the assistant %d  with error %v", assistantId, tx.Error)
		return nil, tx.Error
	}
	return whatsappDeployment, nil
}

func (eService assistantDeploymentService) GetAllAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantApiDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantApiDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			paginate.GetPage(),
			paginate.GetPageSize(),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list api deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) GetAllAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantDebuggerDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantDebuggerDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			paginate.GetPage(),
			paginate.GetPageSize(),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list debugger deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) GetAllAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantPhoneDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantPhoneDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("TelephonyOption").
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			paginate.GetPage(),
			paginate.GetPageSize(),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list phone deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) GetAllAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantWebPluginDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantWebPluginDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("InputAudio", "audio_type = ?", "input").
		Preload("InputAudio.AudioOptions").
		Preload("OutputAudio", "audio_type = ?", "output").
		Preload("OutputAudio.AudioOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			paginate.GetPage(),
			paginate.GetPageSize(),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list webplugin deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) GetAllAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	var (
		deployments []*internal_assistant_entity.AssistantWhatsappDeployment
		cnt         int64
	)
	qry := db.Model(&internal_assistant_entity.AssistantWhatsappDeployment{}).
		Where("assistant_id = ?", assistantId)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Preload("WhatsappOptions").
		Scopes(gorm_models.Paginate(gorm_models.NewPaginated(
			paginate.GetPage(),
			paginate.GetPageSize(),
			&cnt,
			qry,
		))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&deployments)
	if tx.Error != nil {
		eService.logger.Errorf("not able to list whatsapp deployments for assistant %d with error %v", assistantId, tx.Error)
		return cnt, nil, tx.Error
	}
	return cnt, deployments, nil
}

func (eService assistantDeploymentService) DisableAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantApiDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantApiDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantApiDeployment
		getTx := tx.
			Preload("InputAudio", "audio_type = ?", "input").
			Preload("InputAudio.AudioOptions").
			Preload("OutputAudio", "audio_type = ?", "output").
			Preload("OutputAudio.AudioOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantApiDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantApiDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:              current.Greeting,
				GreetingInterruptible: current.GreetingInterruptible,
				Mistake:               current.Mistake,
				UnclearInputTimeout:   current.UnclearInputTimeout,
				UnclearInputMessage:   current.UnclearInputMessage,
				IdleTimeout:           current.IdleTimeout,
				IdleTimeoutBackoff:    current.IdleTimeoutBackoff,
				IdleTimeoutMessage:    current.IdleTimeoutMessage,
				MaxSessionDuration:    current.MaxSessionDuration,
			},
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if current.InputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "input", toProtoAudioProvider(current.InputAudio))
		}
		if current.OutputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "output", toProtoAudioProvider(current.OutputAudio))
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) DisableAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantDebuggerDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantDebuggerDeployment
		getTx := tx.
			Preload("InputAudio", "audio_type = ?", "input").
			Preload("InputAudio.AudioOptions").
			Preload("OutputAudio", "audio_type = ?", "output").
			Preload("OutputAudio.AudioOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantDebuggerDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantDebuggerDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:              current.Greeting,
				GreetingInterruptible: current.GreetingInterruptible,
				Mistake:               current.Mistake,
				UnclearInputTimeout:   current.UnclearInputTimeout,
				UnclearInputMessage:   current.UnclearInputMessage,
				IdleTimeout:           current.IdleTimeout,
				IdleTimeoutBackoff:    current.IdleTimeoutBackoff,
				IdleTimeoutMessage:    current.IdleTimeoutMessage,
				MaxSessionDuration:    current.MaxSessionDuration,
			},
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if current.InputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "input", toProtoAudioProvider(current.InputAudio))
		}
		if current.OutputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "output", toProtoAudioProvider(current.OutputAudio))
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) DisableAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantPhoneDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantPhoneDeployment
		getTx := tx.
			Preload("TelephonyOption").
			Preload("InputAudio", "audio_type = ?", "input").
			Preload("InputAudio.AudioOptions").
			Preload("OutputAudio", "audio_type = ?", "output").
			Preload("OutputAudio.AudioOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantPhoneDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantPhoneDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:              current.Greeting,
				GreetingInterruptible: current.GreetingInterruptible,
				Mistake:               current.Mistake,
				UnclearInputTimeout:   current.UnclearInputTimeout,
				UnclearInputMessage:   current.UnclearInputMessage,
				IdleTimeout:           current.IdleTimeout,
				IdleTimeoutBackoff:    current.IdleTimeoutBackoff,
				IdleTimeoutMessage:    current.IdleTimeoutMessage,
				MaxSessionDuration:    current.MaxSessionDuration,
			},
			AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
				TelephonyProvider: current.TelephonyProvider,
			},
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}

		if current.InputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "input", toProtoAudioProvider(current.InputAudio))
		}
		if current.OutputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "output", toProtoAudioProvider(current.OutputAudio))
		}

		if len(current.TelephonyOption) > 0 {
			phoneOpts := make([]*internal_assistant_entity.AssistantDeploymentTelephonyOption, 0, len(current.TelephonyOption))
			for _, v := range current.TelephonyOption {
				phoneOpts = append(phoneOpts, &internal_assistant_entity.AssistantDeploymentTelephonyOption{
					AssistantDeploymentTelephonyId: created.Id,
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_ACTIVE,
					},
					Metadata: gorm_models.Metadata{
						Key:   v.Key,
						Value: v.Value,
					},
				})
			}
			if err := tx.Create(phoneOpts).Error; err != nil {
				return err
			}
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) DisableAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantWebPluginDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantWebPluginDeployment
		getTx := tx.
			Preload("InputAudio", "audio_type = ?", "input").
			Preload("InputAudio.AudioOptions").
			Preload("OutputAudio", "audio_type = ?", "output").
			Preload("OutputAudio.AudioOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantWebPluginDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantWebPluginDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:              current.Greeting,
				GreetingInterruptible: current.GreetingInterruptible,
				Mistake:               current.Mistake,
				UnclearInputTimeout:   current.UnclearInputTimeout,
				UnclearInputMessage:   current.UnclearInputMessage,
				IdleTimeout:           current.IdleTimeout,
				IdleTimeoutBackoff:    current.IdleTimeoutBackoff,
				IdleTimeoutMessage:    current.IdleTimeoutMessage,
				MaxSessionDuration:    current.MaxSessionDuration,
			},
			Suggestion: current.Suggestion,
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if current.InputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "input", toProtoAudioProvider(current.InputAudio))
		}
		if current.OutputAudio != nil {
			_, _ = eService.createAssistantDeploymentAudio(ctx, auth, created.Id, "output", toProtoAudioProvider(current.OutputAudio))
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) DisableAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	db := eService.postgres.DB(ctx)
	var out *internal_assistant_entity.AssistantWhatsappDeployment
	err := db.Transaction(func(tx *gorm.DB) error {
		var current *internal_assistant_entity.AssistantWhatsappDeployment
		getTx := tx.
			Preload("WhatsappOptions").
			Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_date"}, Desc: true}).
			First(&current)
		if errors.Is(getTx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if getTx.Error != nil {
			return getTx.Error
		}

		if err := eService.archiveDeploymentRecords(ctx, tx, &internal_assistant_entity.AssistantWhatsappDeployment{}, assistantId, *auth.GetUserId()); err != nil {
			return err
		}

		created := &internal_assistant_entity.AssistantWhatsappDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_INACTIVE,
					},
					AssistantId: assistantId,
				},
				Greeting:              current.Greeting,
				GreetingInterruptible: current.GreetingInterruptible,
				Mistake:               current.Mistake,
				UnclearInputTimeout:   current.UnclearInputTimeout,
				UnclearInputMessage:   current.UnclearInputMessage,
				IdleTimeout:           current.IdleTimeout,
				IdleTimeoutBackoff:    current.IdleTimeoutBackoff,
				IdleTimeoutMessage:    current.IdleTimeoutMessage,
				MaxSessionDuration:    current.MaxSessionDuration,
			},
			AssistantDeploymentWhatsapp: internal_assistant_entity.AssistantDeploymentWhatsapp{
				WhatsappProvider: current.WhatsappProvider,
			},
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if len(current.WhatsappOptions) > 0 {
			whatsappOpts := make([]*internal_assistant_entity.AssistantDeploymentWhatsappOption, 0, len(current.WhatsappOptions))
			for _, v := range current.WhatsappOptions {
				whatsappOpts = append(whatsappOpts, &internal_assistant_entity.AssistantDeploymentWhatsappOption{
					AssistantDeploymentWhatsappId: created.Id,
					Mutable: gorm_models.Mutable{
						CreatedBy: *auth.GetUserId(),
						UpdatedBy: *auth.GetUserId(),
						Status:    type_enums.RECORD_ACTIVE,
					},
					Metadata: gorm_models.Metadata{
						Key:   v.Key,
						Value: v.Value,
					},
				})
			}
			if err := tx.Create(whatsappOpts).Error; err != nil {
				return err
			}
		}
		out = created
		return nil
	})
	return out, err
}

func (eService assistantDeploymentService) archiveDeploymentRecords(ctx context.Context, db *gorm.DB, model interface{}, assistantId uint64, userId uint64) error {
	return db.WithContext(ctx).
		Model(model).
		Where("assistant_id = ? AND status IN ?", assistantId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INACTIVE}).
		Updates(map[string]interface{}{
			"status":     type_enums.RECORD_ARCHIEVE,
			"updated_by": userId,
		}).Error
}

func toProtoAudioProvider(audio *internal_assistant_entity.AssistantDeploymentAudio) *protos.DeploymentAudioProvider {
	if audio == nil {
		return nil
	}
	opts := make([]*protos.Metadata, 0, len(audio.AudioOptions))
	for _, v := range audio.AudioOptions {
		opts = append(opts, &protos.Metadata{Key: v.Key, Value: v.Value})
	}
	return &protos.DeploymentAudioProvider{
		AudioProvider: audio.AudioProvider,
		AudioOptions:  opts,
		Status:        string(type_enums.RECORD_ACTIVE),
		AudioType:     audio.AudioType,
	}
}
