package modmgr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
)

func hashModVersion(version ModVersion) (string, error) {
	hasher := sha256.New()
	err := json.MarshalWrite(hasher, version, json.Deterministic(true))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
