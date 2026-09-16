package repomgr

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDetectDoorstopVersion(t *testing.T) {
	tempDir := t.TempDir()

	// Test case: no winhttp.dll
	if ver := DetectDoorstopVersion(tempDir); ver != DoorstopVersionUnknown {
		t.Fatalf("expected unknown for missing file, got %v", ver)
	}

	// Test case: Doorstop 3 UTF-16 signature
	v3Sig := []byte{'-', 0, '-', 0, 'd', 0, 'o', 0, 'o', 0, 'r', 0, 's', 0, 't', 0, 'o', 0, 'p', 0, '-', 0, 'e', 0, 'n', 0, 'a', 0, 'b', 0, 'l', 0, 'e', 0, 0, 0}
	dllPath := filepath.Join(tempDir, "winhttp.dll")
	if err := os.WriteFile(dllPath, v3Sig, 0644); err != nil {
		t.Fatal(err)
	}
	if ver := DetectDoorstopVersion(tempDir); ver != DoorstopVersion3 {
		t.Fatalf("expected DoorstopVersion3, got %v", ver)
	}

	// Test case: Doorstop 4 UTF-16 signature
	v4Sig := []byte{'-', 0, '-', 0, 'd', 0, 'o', 0, 'o', 0, 'r', 0, 's', 0, 't', 0, 'o', 0, 'p', 0, '-', 0, 'e', 0, 'n', 0, 'a', 0, 'b', 0, 'l', 0, 'e', 0, 'd', 0}
	if err := os.WriteFile(dllPath, v4Sig, 0644); err != nil {
		t.Fatal(err)
	}
	if ver := DetectDoorstopVersion(tempDir); ver != DoorstopVersion4 {
		t.Fatalf("expected DoorstopVersion4, got %v", ver)
	}
}

func TestDetectBepInExTarget(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "BepInEx", "core")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Default without files
	target, isIL2CPP := DetectBepInExTarget(tempDir)
	if !strings.HasSuffix(target, "BepInEx.Preloader.dll") || isIL2CPP {
		t.Fatalf("expected BepInEx.Preloader.dll, got %s (isIL2CPP: %v)", target, isIL2CPP)
	}

	// Test IL2CPP detection
	il2cppDll := filepath.Join(coreDir, "BepInEx.Unity.IL2CPP.dll")
	if err := os.WriteFile(il2cppDll, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	target, isIL2CPP = DetectBepInExTarget(tempDir)
	if target != il2cppDll || !isIL2CPP {
		t.Fatalf("expected %s, got %s (isIL2CPP: %v)", il2cppDll, target, isIL2CPP)
	}
}

func TestBuildDoorstopArgs(t *testing.T) {
	tempDir := t.TempDir()
	coreDir := filepath.Join(tempDir, "BepInEx", "core")
	_ = os.MkdirAll(coreDir, 0755)

	// Write Doorstop 3 winhttp.dll
	v3Sig := []byte{'-', 0, '-', 0, 'd', 0, 'o', 0, 'o', 0, 'r', 0, 's', 0, 't', 0, 'o', 0, 'p', 0, '-', 0, 'e', 0, 'n', 0, 'a', 0, 'b', 0, 'l', 0, 'e', 0, 0, 0}
	_ = os.WriteFile(filepath.Join(tempDir, "winhttp.dll"), v3Sig, 0644)

	args, err := BuildDoorstopArgs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(args, "--doorstop-enable") || !slices.Contains(args, "--doorstop-target") {
		t.Fatalf("Doorstop 3 args missing required flags: %v", args)
	}
	if slices.Contains(args, "--doorstop-enabled") || slices.Contains(args, "--doorstop-target-assembly") {
		t.Fatalf("Doorstop 3 should not have Doorstop 4 flags: %v", args)
	}

	// Write Doorstop 4 winhttp.dll
	v4Sig := []byte{'-', 0, '-', 0, 'd', 0, 'o', 0, 'o', 0, 'r', 0, 's', 0, 't', 0, 'o', 0, 'p', 0, '-', 0, 'e', 0, 'n', 0, 'a', 0, 'b', 0, 'l', 0, 'e', 0, 'd', 0}
	_ = os.WriteFile(filepath.Join(tempDir, "winhttp.dll"), v4Sig, 0644)

	args, err = BuildDoorstopArgs(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(args, "--doorstop-enabled") || !slices.Contains(args, "--doorstop-target-assembly") {
		t.Fatalf("Doorstop 4 args missing required flags: %v", args)
	}
	if slices.Contains(args, "--doorstop-enable") || slices.Contains(args, "--doorstop-target") {
		t.Fatalf("Doorstop 4 should not have Doorstop 3 flags: %v", args)
	}
}

func TestGenerateDoorstopConfig(t *testing.T) {
	config := GenerateDoorstopConfig(`C:\fake\profile`)

	// Must contain Doorstop 3 section
	if !strings.Contains(config, "[UnityDoorstop]") || !strings.Contains(config, "targetAssembly =") {
		t.Fatalf("Doorstop 3 config missing: %s", config)
	}

	// Must contain Doorstop 4 section
	if !strings.Contains(config, "[General]") || !strings.Contains(config, "target_assembly =") {
		t.Fatalf("Doorstop 4 config missing: %s", config)
	}
}
