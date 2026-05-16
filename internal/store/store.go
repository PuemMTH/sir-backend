package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/syumai/workers/cloudflare/d1"

	"github.com/sir-labs/sir-api/internal/model"
)

// Store wraps a GORM DB and provides content data-access operations.
type Store struct {
	db *gorm.DB
}

// Open creates a new Store backed by the "DB" D1 binding.
func Open() (*Store, error) {
	connector, err := d1.OpenConnector("DB")
	if err != nil {
		return nil, err
	}
	sqlDB := sql.OpenDB(connector)
	gormDB, err := gorm.Open(newDialector(sqlDB), &gorm.Config{
		Logger:                 defaultLogger(),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		sqlDB.Close()
		return nil, err
	}
	return &Store{db: gormDB}, nil
}

func defaultLogger() gormlogger.Interface {
	return gormlogger.Default.LogMode(gormlogger.Silent)
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ── Notes ────────────────────────────────────────────────────────────────────

func (s *Store) CreateNote(ctx context.Context, userID, title, content string) (*model.Note, error) {
	n := &model.Note{
		UserID:  userID,
		Title:   title,
		Content: content,
	}
	if err := s.db.WithContext(ctx).Create(n).Error; err != nil {
		return nil, err
	}
	return n, nil
}

func (s *Store) ListNotes(ctx context.Context, userID string) ([]model.Note, error) {
	var notes []model.Note
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&notes).Error
	return notes, err
}

func (s *Store) GetNote(ctx context.Context, userID, noteID string) (*model.Note, error) {
	var n model.Note
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", noteID, userID).
		Take(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &n, err
}

func (s *Store) DeleteNote(ctx context.Context, userID, noteID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", noteID, userID).
		Delete(&model.Note{}).Error
}

// ── LaTeX Files ───────────────────────────────────────────────────────────────

func (s *Store) CreateLatexFile(ctx context.Context, id, userID, name, r2Key, engine string) (*model.LatexFile, error) {
	f := &model.LatexFile{
		ID:     id,
		UserID: userID,
		Name:   name,
		R2Key:  r2Key,
		Engine: engine,
	}
	if err := s.db.WithContext(ctx).Create(f).Error; err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Store) ListLatexFiles(ctx context.Context, userID string) ([]model.LatexFile, error) {
	var files []model.LatexFile
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&files).Error
	return files, err
}

func (s *Store) GetLatexFile(ctx context.Context, userID, fileID string) (*model.LatexFile, error) {
	var f model.LatexFile
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", fileID, userID).
		Take(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &f, err
}

func (s *Store) UpdateLatexFile(ctx context.Context, f *model.LatexFile) (*model.LatexFile, error) {
	now := time.Now().Unix()
	err := s.db.WithContext(ctx).
		Model(&model.LatexFile{}).
		Where("id = ? AND user_id = ?", f.ID, f.UserID).
		Updates(map[string]any{
			"name":       f.Name,
			"engine":     f.Engine,
			"updated_at": now,
		}).Error
	if err != nil {
		return nil, err
	}
	f.UpdatedAt = now
	return f, nil
}

func (s *Store) DeleteLatexFile(ctx context.Context, userID, fileID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", fileID, userID).
		Delete(&model.LatexFile{}).Error
}

// ── User Assets ───────────────────────────────────────────────────────────────

func (s *Store) CreateUserAsset(ctx context.Context, id, userID, name, r2Key, thumbnailR2Key, mimeType string, size int64) (*model.UserAsset, error) {
	a := &model.UserAsset{
		ID:             id,
		UserID:         userID,
		Name:           name,
		R2Key:          r2Key,
		ThumbnailR2Key: thumbnailR2Key,
		MimeType:       mimeType,
		Size:           size,
	}
	if err := s.db.WithContext(ctx).Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Store) ListUserAssets(ctx context.Context, userID string) ([]model.UserAsset, error) {
	var assets []model.UserAsset
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&assets).Error
	return assets, err
}

func (s *Store) GetUserAsset(ctx context.Context, userID, assetID string) (*model.UserAsset, error) {
	var a model.UserAsset
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", assetID, userID).
		Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (s *Store) DeleteUserAsset(ctx context.Context, userID, assetID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", assetID, userID).
		Delete(&model.UserAsset{}).Error
}

// ── PDF Cache ────────────────────────────────────────────────────────────────

func (s *Store) GetPDFCache(ctx context.Context, sourceHash string) (*model.PDFCache, error) {
	var c model.PDFCache
	err := s.db.WithContext(ctx).Where("source_hash = ?", sourceHash).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (s *Store) CreatePDFCache(ctx context.Context, sourceHash, r2Key, engine string) error {
	c := &model.PDFCache{
		SourceHash: sourceHash,
		R2Key:      r2Key,
		Engine:     engine,
	}
	err := s.db.WithContext(ctx).Create(c).Error
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return nil
	}
	return err
}
