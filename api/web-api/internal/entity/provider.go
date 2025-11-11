package internal_entity

// type ToolProvider struct {
// 	Id                   uint64                  `json:"id" gorm:"type:bigint;primaryKey;<-:create"`
// 	Name                 string                  `json:"name" gorm:"type:string;size:200;not null"`
// 	Description          string                  `json:"description" gorm:"type:string;size:200;not null"`
// 	Image                string                  `json:"image" gorm:"type:string;size:200;not null"`
// 	Feature              gorm_types.StringArray  `json:"feature" gorm:"type:string;size:200;not null"`
// 	ConnectConfiguration gorm_types.InterfaceMap `json:"connectConfiguration" gorm:"type:string;size:200;not null"`
// }

// type Provider struct {
// 	gorm_model.Audited
// 	Name                 string              `json:"name" gorm:"type:string;size:200;not null;index:pr_idx_name"`
// 	Description          string              `json:"description" gorm:"type:string;size:400;not null"`
// 	HumanName            string              `json:"humanName" gorm:"type:string;size:100;not null;unique:pr_unq_human_name"`
// 	Website              string              `json:"website" gorm:"type:string;size:250;not null"`
// 	Image                string              `json:"image" gorm:"type:string;size:250;not null"`
// 	Status               string              `json:"status" gorm:"type:string;size:50;not null;default:active"`
// 	ConnectConfiguration gorm_types.MapArray `json:"connectConfiguration" gorm:"type:string;size:200;not null"`
// }

// type ProviderModel struct {
// 	gorm_model.Audited
// 	ProviderId  uint64                   `json:"providerId" gorm:"type:bigint;size:200;not null;index:prm_idx_provider"`
// 	Name        string                   `json:"name" gorm:"type:string;size:200;not null;index:prm_idx_name"`
// 	Description string                   `json:"description" gorm:"type:string;size:400;not null"`
// 	HumanName   string                   `json:"humanName" gorm:"type:string;size:100;not null"`
// 	Category    string                   `json:"category" gorm:"type:string;size:250"`
// 	Status      string                   `json:"status" gorm:"type:string;size:50;not null;default:active"`
// 	Owner       string                   `json:"owner" gorm:"type:string;size:50;not null;default:rapida"`
// 	Parameters  []*ProviderModelVariable `gorm:"foreignKey:ProviderModelId"`
// 	Provider    Provider                 `gorm:"foreignKey:ProviderId"`
// 	Metadatas   []*ProviderModelMetadata `gorm:"foreignKey:ProviderModelId"`
// 	Endpoint    string                   `json:"endpoint"`
// }

// type ProviderModelEndpoint struct {
// 	gorm_model.Audited
// 	ProviderModelId uint64 `json:"providerModelId" gorm:"type:bigint;not null"`
// 	Endpoint        string `json:"endpoint" gorm:"type:string;size:200;not null"`
// }

// type ProviderModelMetadata struct {
// 	gorm_model.Audited
// 	gorm_model.Metadata
// 	ProviderModelId uint64 `json:"providerModelId" gorm:"type:bigint;not null"`
// 	Name            string `json:"name" gorm:"type:string;size:200;not null"`
// }

// type ProviderModelVariable struct {
// 	gorm_model.Audited
// 	ProviderModelId uint64                          `json:"provider_model_id" gorm:"type:bigint;size:200;not null;index:pmv_idx_provider"`
// 	Type            string                          `json:"type" gorm:"type:string;size:200;not null"`
// 	Place           string                          `json:"place" gorm:"type:string;size:200;not null"`
// 	Name            string                          `json:"name" gorm:"type:string;size:200;not null"`
// 	Key             string                          `json:"key" gorm:"type:string;size:200;not null"`
// 	Description     string                          `json:"description" gorm:"type:string;size:200;not null"`
// 	DefaultValue    string                          `json:"defaultValue" gorm:"type:string;size:200;not null"`
// 	Metadatas       []ProviderModelVariableMetadata `gorm:"foreignKey:ProviderModelVariableId"`
// }

// type ProviderModelVariableMetadata struct {
// 	gorm_model.Audited
// 	gorm_model.Metadata
// 	ProviderModelVariableId uint64 `json:"ProviderModelVariableId" gorm:"type:bigint;not null"`
// }
