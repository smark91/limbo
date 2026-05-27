package database

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"gorm.io/gorm"
)

const (
	VapidPublicKeyMetadataKey  = "vapid_public_key"
	VapidPrivateKeyMetadataKey = "vapid_private_key"
)

// GetOrGenerateVapidKeys loads VAPID keys from the database or generates them if not present.
func GetOrGenerateVapidKeys(db *gorm.DB) (string, string, error) {
	var pubKeyMeta, privKeyMeta SystemMetadata

	errPub := db.Where("key = ?", VapidPublicKeyMetadataKey).First(&pubKeyMeta).Error
	errPriv := db.Where("key = ?", VapidPrivateKeyMetadataKey).First(&privKeyMeta).Error

	if errPub == nil && errPriv == nil {
		slog.Debug("[DB] Loaded existing VAPID keys from database")
		return pubKeyMeta.Value, privKeyMeta.Value, nil
	}

	if errPub != nil && !errors.Is(errPub, gorm.ErrRecordNotFound) {
		return "", "", fmt.Errorf("failed to load VAPID public key from db: %w", errPub)
	}
	if errPriv != nil && !errors.Is(errPriv, gorm.ErrRecordNotFound) {
		return "", "", fmt.Errorf("failed to load VAPID private key from db: %w", errPriv)
	}

	slog.Info("VAPID keys not found in database. Generating new key pair...")
	privKey, pubKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate VAPID keys: %w", err)
	}

	now := time.Now()
	err = db.Transaction(func(tx *gorm.DB) error {
		err := tx.Save(&SystemMetadata{
			Key:       VapidPublicKeyMetadataKey,
			Value:     pubKey,
			UpdatedAt: now,
		}).Error
		if err != nil {
			return err
		}

		return tx.Save(&SystemMetadata{
			Key:       VapidPrivateKeyMetadataKey,
			Value:     privKey,
			UpdatedAt: now,
		}).Error
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to save generated VAPID keys: %w", err)
	}

	slog.Info("Successfully generated and saved new VAPID keys to database")
	return pubKey, privKey, nil
}
