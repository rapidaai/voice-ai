package gorm_models

type ProviderModelParameter struct {
	Audited
	ProviderModelVariableId uint64 `json:"providerModelVariableId" gorm:"type:bigint;not null"`
	Value                   string `json:"value" gorm:"type:string;size:200;not null"`
}
