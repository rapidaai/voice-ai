package gorm_models

type Metric struct {
	Name        string `json:"name" gorm:"type:text"`
	Value       string `json:"value" gorm:"type:text"`
	Description string `json:"description" gorm:"type:text"`
}
