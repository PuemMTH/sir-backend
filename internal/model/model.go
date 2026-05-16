package model

import (
	"github.com/sir-labs/sir-api/internal/token"
	"gorm.io/gorm"
)

// Note represents a private note owned by a user.
type Note struct {
	ID        string `gorm:"column:id;primaryKey"`
	UserID    string `gorm:"column:user_id;not null;index"`
	Title     string `gorm:"column:title;not null"`
	Content   string `gorm:"column:content;not null"`
	CreatedAt int64  `gorm:"column:created_at;autoCreateTime:unix"`
	UpdatedAt int64  `gorm:"column:updated_at;autoUpdateTime:unix"`
}

func (n *Note) BeforeCreate(_ *gorm.DB) error {
	if n.ID == "" {
		id, err := token.RandomString(12)
		if err != nil {
			return err
		}
		n.ID = id
	}
	return nil
}

// LatexFile represents a LaTeX source file owned by a user, stored in R2.
type LatexFile struct {
	ID        string `gorm:"column:id;primaryKey" json:"id"`
	UserID    string `gorm:"column:user_id;not null;index" json:"user_id"`
	Name      string `gorm:"column:name;not null" json:"name"`
	R2Key     string `gorm:"column:r2_key;not null" json:"r2_key"`
	Engine    string `gorm:"column:engine;not null;default:lualatex" json:"engine"`
	CreatedAt int64  `gorm:"column:created_at;autoCreateTime:unix" json:"created_at"`
	UpdatedAt int64  `gorm:"column:updated_at;autoUpdateTime:unix" json:"updated_at"`
}

func (f *LatexFile) BeforeCreate(_ *gorm.DB) error {
	if f.ID == "" {
		id, err := token.RandomString(12)
		if err != nil {
			return err
		}
		f.ID = id
	}
	return nil
}

// PDFCache records a compiled PDF's R2 location keyed by MD5(engine+":"+source).
type PDFCache struct {
	SourceHash string `gorm:"column:source_hash;primaryKey" json:"source_hash"`
	R2Key      string `gorm:"column:r2_key;not null"         json:"r2_key"`
	Engine     string `gorm:"column:engine;not null"         json:"engine"`
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:unix" json:"created_at"`
}

func (PDFCache) TableName() string { return "pdf_cache" }

// UserAsset represents an uploaded file (image, font, etc.) owned by a user, stored in R2.
type UserAsset struct {
	ID             string `gorm:"column:id;primaryKey" json:"id"`
	UserID         string `gorm:"column:user_id;not null;index" json:"user_id"`
	Name           string `gorm:"column:name;not null" json:"name"`
	R2Key          string `gorm:"column:r2_key;not null" json:"r2_key"`
	ThumbnailR2Key string `gorm:"column:thumbnail_r2_key" json:"thumbnail_r2_key,omitempty"`
	MimeType       string `gorm:"column:mime_type;not null;default:application/octet-stream" json:"mime_type"`
	Size           int64  `gorm:"column:size;not null;default:0" json:"size"`
	CreatedAt      int64  `gorm:"column:created_at;autoCreateTime:unix" json:"created_at"`
}

func (a *UserAsset) TableName() string { return "user_assets" }

func (a *UserAsset) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		id, err := token.RandomString(12)
		if err != nil {
			return err
		}
		a.ID = id
	}
	return nil
}

// Setting is a key/value system configuration entry.
type Setting struct {
	Key       string `gorm:"column:key;primaryKey"          json:"key"`
	Value     string `gorm:"column:value;not null"          json:"value"`
	UpdatedAt int64  `gorm:"column:updated_at;autoUpdateTime:unix" json:"updated_at"`
}
