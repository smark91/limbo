package database

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetOrGenerateVapidKeys(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&SystemMetadata{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// 1. First invocation: keys should be generated and returned
	pubKey1, privKey1, err := GetOrGenerateVapidKeys(db)
	if err != nil {
		t.Fatalf("expected no error generating keys, got: %v", err)
	}

	if pubKey1 == "" || privKey1 == "" {
		t.Fatalf("expected keys to be non-empty, got public=%q private=%q", pubKey1, privKey1)
	}

	// Verify database contains the keys
	var pubMeta, privMeta SystemMetadata
	if err := db.Where("key = ?", VapidPublicKeyMetadataKey).First(&pubMeta).Error; err != nil {
		t.Errorf("failed to find public key in metadata table: %v", err)
	}
	if err := db.Where("key = ?", VapidPrivateKeyMetadataKey).First(&privMeta).Error; err != nil {
		t.Errorf("failed to find private key in metadata table: %v", err)
	}

	if pubMeta.Value != pubKey1 || privMeta.Value != privKey1 {
		t.Errorf("expected db stored keys to match returned ones")
	}

	// 2. Second invocation: should load keys from database, not re-generate
	pubKey2, privKey2, err := GetOrGenerateVapidKeys(db)
	if err != nil {
		t.Fatalf("expected no error loading keys, got: %v", err)
	}

	if pubKey2 != pubKey1 || privKey2 != privKey1 {
		t.Fatalf("expected loaded keys to be identical to generated keys, got: pub=%q (expected %q), priv=%q (expected %q)", pubKey2, pubKey1, privKey2, privKey1)
	}
}
