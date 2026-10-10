package model

type FieldType string

const (
	FieldTypeText   FieldType = "text"
	FieldTypeChoice FieldType = "choice"
	FieldTypeScale  FieldType = "scale"
)

const DefaultPluginSlug = "freetext"

type DiaryPluginMeta struct {
	Slug        string
	Name        string
	Description string
	Icon        string
}

type FieldSpec struct {
	Name     string
	Prompt   string
	Type     FieldType
	Options  []string
	Min      int
	Max      int
	Required bool
	Multiple bool
}

type EntryField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
