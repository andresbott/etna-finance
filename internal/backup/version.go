package backup

const (
	SchemaV1 = "1.0.0"
)

type metaInfoV1 struct {
	Version string `json:"version"`
	Date    string `json:"date"`
	Tenant  string `json:"tenant"`
}

type AccountProviderV1 struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
type AccountV1 struct {
	ID                uint   `json:"id"`
	AccountProviderID uint   `json:"providerId"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Currency          string `json:"currency"`
	Type              string `json:"accountType"`
}

type CategoryV1 struct {
	ID          uint   `json:"id"`
	ParentId    uint   `json:"ParentId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"categoryType"`
}
